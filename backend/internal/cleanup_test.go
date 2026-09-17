package internal

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestRunCleanupClearsEmailsOnlyAfterEvent(t *testing.T) {
	db := testDB(t)

	cases := []struct {
		slug      string
		eventDate time.Time
		cleared   bool
	}{
		{"twodaysago", time.Now().Add(-48 * time.Hour), true},
		{"thismorning", time.Now().Add(-12 * time.Hour), false}, // may not be over in the organizer's timezone
		{"tomorrow", time.Now().Add(24 * time.Hour), false},
	}
	for _, c := range cases {
		insertRemindMe(t, db, insertEvent(t, db, c.slug, c.eventDate), c.slug)
	}

	res := RunCleanup(db)
	if res.EventsDeleted != 0 {
		t.Errorf("deleted %d events, want 0 (none past retention)", res.EventsDeleted)
	}
	if res.EmailsCleared != 1 {
		t.Errorf("cleared %d emails, want 1", res.EmailsCleared)
	}

	for _, c := range cases {
		var hasEmail bool
		if err := db.QueryRow(`
			SELECT r.notify_via IS NOT NULL FROM responses r JOIN events e ON e.id = r.event_id
			WHERE e.slug = $1
		`, c.slug).Scan(&hasEmail); err != nil {
			t.Fatal(err)
		}
		if hasEmail == c.cleared {
			t.Errorf("%s: email present = %v, want %v", c.slug, hasEmail, !c.cleared)
		}
	}
}

func TestRunCleanupKeepsEventsWithinRetention(t *testing.T) {
	db := testDB(t)
	insertEvent(t, db, "old", time.Now().AddDate(0, -3, 0))
	insertEvent(t, db, "recent", time.Now().AddDate(0, -1, 0))
	insertEvent(t, db, "undated", time.Time{})
	db.Exec(`UPDATE events SET event_date = NULL WHERE slug = 'undated'`)

	res := RunCleanup(db) // default retention: 2 months
	if res.EventsDeleted != 1 {
		t.Errorf("deleted %d events, want 1", res.EventsDeleted)
	}
	var left []string
	rows, _ := db.Query(`SELECT slug FROM events ORDER BY slug`)
	for rows.Next() {
		var s string
		rows.Scan(&s)
		left = append(left, s)
	}
	rows.Close()
	if len(left) != 2 || left[0] != "recent" || left[1] != "undated" {
		t.Errorf("events left = %v, want [recent undated]", left)
	}
}

func TestRunCleanupDeletesDemoEventsAfterThreeDays(t *testing.T) {
	db := testDB(t)
	future := time.Now().Add(7 * 24 * time.Hour)
	for _, e := range []struct {
		slug    string
		demo    bool
		created time.Duration // ago
	}{
		{"olddemo", true, 73 * time.Hour},
		{"oldreal", false, 73 * time.Hour},
		{"newdemo", true, 71 * time.Hour},
	} {
		insertEvent(t, db, e.slug, future)
		if _, err := db.Exec(`UPDATE events SET is_demo = $1, created_at = now() - $2::interval WHERE slug = $3`,
			e.demo, fmt.Sprintf("%d seconds", int(e.created.Seconds())), e.slug); err != nil {
			t.Fatal(err)
		}
	}

	res := RunCleanup(db)
	if res.DemoEventsDeleted != 1 {
		t.Errorf("deleted %d demo events, want 1", res.DemoEventsDeleted)
	}
	var left []string
	rows, err := db.Query(`SELECT slug FROM events ORDER BY slug`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var s string
		rows.Scan(&s)
		left = append(left, s)
	}
	if strings.Join(left, ",") != "newdemo,oldreal" {
		t.Errorf("events left = %v, want [newdemo oldreal]", left)
	}
}
