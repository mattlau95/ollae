package internal

import (
	"strings"
	"testing"
)

func TestCleanField(t *testing.T) {
	got, err := cleanField("  Alex K.  ", maxNameLen, "Names")
	if err != nil || got != "Alex K." {
		t.Errorf("cleanField trims: got %q, %v", got, err)
	}
	if _, err := cleanField(strings.Repeat("é", maxNameLen), maxNameLen, "Names"); err != nil {
		t.Errorf("exactly %d characters (multi-byte) should pass: %v", maxNameLen, err)
	}
	_, err = cleanField(strings.Repeat("a", maxNameLen+1), maxNameLen, "Names")
	if err == nil || err.Error() != "Names can be up to 40 characters." {
		t.Errorf("too long: got %v", err)
	}
	if _, err := cleanField("big blockedword energy", maxTitleLen, "Titles"); err == nil {
		t.Error("blocked word should be rejected")
	}
}

func TestIsBlocked(t *testing.T) {
	blocked := []string{
		"blockedword", "BLOCKEDWORD", "b.l.o.c.k.e.d.w.o.r.d this", "bl0ckedw0rd", "Bl0ckedword",
		"blockedwords", "xxblockedwordxx", "$illyterm",
	}
	allowed := []string{
		"Blockedwordsmith Street", "Harbor Park", "Riverside United", "Matt",
		"Orchard Street", "Book Club", "Weekly practice", "Board game night",
		"Alexander Library", "Class of 2026", "Spring social", "Oak Trail",
	}
	for _, s := range blocked {
		if !isBlocked(s) {
			t.Errorf("isBlocked(%q) = false, want true", s)
		}
	}
	for _, s := range allowed {
		if isBlocked(s) {
			t.Errorf("isBlocked(%q) = true, want false", s)
		}
	}
}

func TestValidEmoji(t *testing.T) {
	valid := []string{"🏐", "🎉", "❤️", "👨‍👩‍👧", "🇰🇷", "👍🏽", "☕", "#️⃣"}
	invalid := []string{"", "🏐🎉", "a", "7", "ab", "🎉 party", " ", "©"}
	for _, s := range valid {
		if !validEmoji(s) {
			t.Errorf("validEmoji(%q) = false, want true", s)
		}
	}
	for _, s := range invalid {
		if validEmoji(s) {
			t.Errorf("validEmoji(%q) = true, want false", s)
		}
	}
}

func TestValidEmailDateClock(t *testing.T) {
	if !validEmail("a.b@example.co") || validEmail("nope") || validEmail("a b@c.d") || validEmail(strings.Repeat("a", 250)+"@b.co") {
		t.Error("validEmail")
	}
	if !validDate("2026-09-17") || validDate("2026-13-01") || validDate("Saturday") {
		t.Error("validDate")
	}
	if !validClock("12:30") || validClock("24:00") || validClock("12:30pm") {
		t.Error("validClock")
	}
	if !validEventDate("2026-09-17T12:30:00") || validEventDate("2026-09-17") {
		t.Error("validEventDate")
	}
}
