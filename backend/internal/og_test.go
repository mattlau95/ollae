package internal

import (
	"testing"
	"time"
)

func TestIsCrawlerUA(t *testing.T) {
	cases := map[string]bool{
		"facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)": true,
		"WhatsApp/2.23.20.0":       true,
		"Mozilla/5.0 (compatible; Slackbot-LinkExpanding 1.0)": true,
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) Safari/604.1": false,
		"": false,
	}
	for ua, want := range cases {
		if got := isCrawlerUA(ua); got != want {
			t.Errorf("isCrawlerUA(%q) = %v, want %v", ua, got, want)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("hello", 10); got != "hello" {
		t.Errorf("short string changed: %q", got)
	}
	if got := truncate("héllo wörld", 5); got != "héllo..." {
		t.Errorf("rune-safe truncation: got %q", got)
	}
}

func TestEscapeHTML(t *testing.T) {
	in := `<a href="x">&</a>`
	want := `&lt;a href=&quot;x&quot;&gt;&amp;&lt;/a&gt;`
	if got := escapeHTML(in); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatOGWhen(t *testing.T) {
	if got := formatOGWhen(Event{}); got != "" {
		t.Errorf("no date should be empty, got %q", got)
	}
	allDay := time.Date(2026, 6, 6, 0, 0, 0, 0, time.UTC)
	if got := formatOGWhen(Event{EventDate: &allDay}); got != "Sat, Jun 6" {
		t.Errorf("all-day: got %q", got)
	}
	timed := time.Date(2026, 6, 6, 14, 30, 0, 0, time.UTC)
	if got := formatOGWhen(Event{EventDate: &timed}); got != "Sat, Jun 6 · 2:30 PM" {
		t.Errorf("timed: got %q", got)
	}
}
