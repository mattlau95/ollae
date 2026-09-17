package internal

import (
	"context"
	"database/sql"
	"log"
	"strings"
	"sync"
	"time"
	"unicode"
)

// Blocklist holds the words and phrases visitors can't use in names,
// titles and locations. The terms live only in the blocked_terms table,
// edited from /admin, so none appear in this public repo. Every machine
// loads them at startup and reloads when blocked_terms_version changes.
//
// Matching works on words, after lowercasing and undoing common character
// swaps (0→o, 1→i, 3→e, 4→a, 5→s, 7→t, @→a, $→s, !→i). Letters spelled
// out one at a time ("t.e.r.m", "t e r m") are joined first. A term inside
// a longer word doesn't match, so place names that contain one pass.
type Blocklist struct {
	db      *sql.DB
	mu      sync.RWMutex
	terms   [][]string // each term as a sequence of normalized words
	version string
}

func NewBlocklist(db *sql.DB) *Blocklist {
	b := &Blocklist{db: db}
	if err := b.Refresh(context.Background()); err != nil {
		log.Printf("blocklist: initial load failed: %v", err)
	}
	return b
}

// Refresh reloads the terms if the version has changed since the last load.
func (b *Blocklist) Refresh(ctx context.Context) error {
	var version string
	err := b.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = 'blocked_terms_version'`).Scan(&version)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	b.mu.RLock()
	loaded := b.terms != nil && version == b.version
	b.mu.RUnlock()
	if loaded {
		return nil
	}

	rows, err := b.db.QueryContext(ctx, `SELECT term FROM blocked_terms`)
	if err != nil {
		return err
	}
	defer rows.Close()
	terms := [][]string{}
	for rows.Next() {
		var term string
		if err := rows.Scan(&term); err != nil {
			return err
		}
		if words := normalizeWords(term); len(words) > 0 {
			terms = append(terms, words)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	b.mu.Lock()
	b.terms, b.version = terms, version
	b.mu.Unlock()
	return nil
}

// StartRefresher checks for changes made on another machine.
func (b *Blocklist) StartRefresher(every time.Duration) {
	go func() {
		ticker := time.NewTicker(every)
		defer ticker.Stop()
		for range ticker.C {
			if err := b.Refresh(context.Background()); err != nil {
				log.Printf("blocklist: refresh failed: %v", err)
			}
		}
	}()
}

// Matches reports whether text contains a blocked term. A nil or empty
// Blocklist lets everything through.
func (b *Blocklist) Matches(text string) bool {
	if b == nil {
		return false
	}
	b.mu.RLock()
	terms := b.terms
	b.mu.RUnlock()
	if len(terms) == 0 {
		return false
	}
	words := normalizeWords(text)
	for _, term := range terms {
		for i := 0; i+len(term) <= len(words); i++ {
			if equalWords(words[i:i+len(term)], term) {
				return true
			}
		}
	}
	return false
}

func equalWords(a, b []string) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

var swaps = map[rune]rune{'0': 'o', '1': 'i', '3': 'e', '4': 'a', '5': 's', '7': 't', '@': 'a', '$': 's', '!': 'i'}

// normalizeWords lowercases text, undoes character swaps and splits it into
// words. The swap characters count as part of a word, except a trailing
// "!", which is punctuation. Runs of three or more single letters are
// joined into one word.
func normalizeWords(text string) []string {
	isWordChar := func(r rune) bool { _, swap := swaps[r]; return unicode.IsLetter(r) || swap }
	var words []string
	for _, chunk := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !isWordChar(r) }) {
		chunk = strings.TrimRight(chunk, "!")
		var sb strings.Builder
		for _, r := range chunk {
			if s, ok := swaps[r]; ok {
				r = s
			}
			sb.WriteRune(r)
		}
		if w := sb.String(); w != "" {
			words = append(words, w)
		}
	}

	var joined []string
	for i := 0; i < len(words); {
		j := i
		for j < len(words) && len([]rune(words[j])) == 1 {
			j++
		}
		if j-i >= 3 {
			joined = append(joined, strings.Join(words[i:j], ""))
			i = j
			continue
		}
		joined = append(joined, words[i])
		i++
	}
	return joined
}

// NormalizeTerm is how a term is stored: its normalized words, space-separated.
func NormalizeTerm(term string) string {
	return strings.Join(normalizeWords(term), " ")
}
