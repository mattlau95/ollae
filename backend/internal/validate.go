package internal

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/rivo/uniseg"
)

const (
	maxNameLen     = 40
	maxTitleLen    = 80
	maxLocationLen = 120
	maxEmailLen    = 254
	maxParseInput  = 500
	maxGuests      = 20
)

// validationError carries a message safe to show the visitor as-is.
type validationError struct{ msg string }

func (e validationError) Error() string { return e.msg }

// cleanField trims s and checks its length and the blocklist. label is how
// the field is named in the message, e.g. "Names".
func cleanField(s string, max int, label string) (string, error) {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > max {
		return "", validationError{fmt.Sprintf("%s can be up to %d characters.", label, max)}
	}
	if isBlocked(s) {
		return "", validationError{"Let's keep it friendly. Try different wording."}
	}
	return s, nil
}

// truncateRunes cuts s to at most max characters, for fields Claude produced
// that the visitor will review before saving.
func truncateRunes(s string, max int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= max {
		return string(r)
	}
	return strings.TrimSpace(string(r[:max]))
}

var emailShape = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

func validEmail(s string) bool {
	return len(s) <= maxEmailLen && emailShape.MatchString(s)
}

// validEmoji reports whether s is exactly one emoji: one grapheme cluster
// that is pictographic, not a letter, digit or symbol that merely renders
// as text.
func validEmoji(s string) bool {
	if s == "" || uniseg.GraphemeClusterCount(s) != 1 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 0x1F000, // emoticons, symbols & pictographs, flags, …
			r >= 0x2190 && r <= 0x2BFF, // arrows, misc technical, dingbats, …
			r == 0x3030, r == 0x303D, r == 0x3297, r == 0x3299,
			r == 0xFE0F, r == 0x20E3: // emoji presentation, keycap
			return true
		}
	}
	return false
}

func validDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

func validClock(s string) bool {
	_, err := time.Parse("15:04", s)
	return err == nil
}

func validEventDate(s string) bool {
	_, err := time.Parse("2006-01-02T15:04:05", s)
	return err == nil
}

// Blocked words are matched as whole words after normalizing, so "Dickens"
// and longer words that merely contain one pass. The few in blockedAnywhere are
// unambiguous enough to catch inside other words too.
var (
	blockedWords = []string{
	}
	blockedAnywhere = []string{}
	leet            = strings.NewReplacer("0", "o", "1", "i", "3", "e", "4", "a", "5", "s", "7", "t", "@", "a", "$", "s", "!", "i")
	blockedSet      = map[string]bool{}
)

func init() {
	for _, w := range blockedWords {
		blockedSet[w] = true
	}
}

func isBlocked(s string) bool {
	norm := leet.Replace(strings.ToLower(s))
	words := strings.FieldsFunc(norm, func(r rune) bool { return !unicode.IsLetter(r) })
	for _, w := range words {
		if blockedSet[w] || blockedSet[strings.TrimSuffix(w, "s")] || blockedSet[strings.TrimSuffix(w, "es")] {
			return true
		}
	}
	letters := strings.Join(words, "")
	for _, w := range blockedAnywhere {
		if strings.Contains(letters, w) {
			return true
		}
	}
	return false
}
