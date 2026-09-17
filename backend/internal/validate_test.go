package internal

import (
	"strings"
	"testing"
)

func TestCleanField(t *testing.T) {
	got, err := cleanField("  Alex K.  ", maxNameLen, "Names", nil)
	if err != nil || got != "Alex K." {
		t.Errorf("cleanField trims: got %q, %v", got, err)
	}
	if _, err := cleanField(strings.Repeat("é", maxNameLen), maxNameLen, "Names", nil); err != nil {
		t.Errorf("exactly %d characters (multi-byte) should pass: %v", maxNameLen, err)
	}
	_, err = cleanField(strings.Repeat("a", maxNameLen+1), maxNameLen, "Names", nil)
	if err == nil || err.Error() != "Names can be up to 40 characters." {
		t.Errorf("too long: got %v", err)
	}
	if _, err := cleanField("big bl0ckedword energy", maxTitleLen, "Titles", staticBlocklist("blockedword")); err == nil {
		t.Error("blocked word should be rejected")
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
