package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeResend stands in for the batch endpoint, recording every recipient.
func fakeResend(t *testing.T, status int) (recipients func() []string) {
	t.Helper()
	var mu sync.Mutex
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var emails []struct {
			To   []string `json:"to"`
			HTML string   `json:"html"`
		}
		if err := json.Unmarshal(body, &emails); err != nil {
			t.Errorf("batch body is not a JSON array: %v", err)
		}
		mu.Lock()
		for _, e := range emails {
			got = append(got, e.To...)
		}
		mu.Unlock()
		w.WriteHeader(status)
		w.Write([]byte(`{"data":[]}`))
	}))
	t.Cleanup(srv.Close)

	oldURL := resendBatchURL
	resendBatchURL = srv.URL
	t.Cleanup(func() { resendBatchURL = oldURL })

	return func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), got...)
	}
}

func TestClaimRemindersNeverOverlaps(t *testing.T) {
	db := testDB(t)
	eventID := insertEvent(t, db, "claimtest", time.Now().Add(24*time.Hour))
	if _, err := db.Exec(`
		INSERT INTO responses (event_id, name, status, notify_via)
		SELECT $1, 'guest' || g, 'remind_me', 'guest' || g || '@example.com'
		FROM generate_series(1, 400) g
	`, eventID); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	seen := map[string]int{}
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			batch, err := claimReminders(db, 25)
			if err != nil {
				t.Error(err)
				return
			}
			mu.Lock()
			for _, rem := range batch {
				seen[rem.ResponseID]++
			}
			mu.Unlock()
		}()
	}
	close(start)
	wg.Wait()

	// SKIP LOCKED lets a claimer come back short, so the total can be under
	// 400; what matters is that every row marked claimed went to exactly one.
	var marked int
	db.QueryRow(`SELECT count(*) FROM responses WHERE reminded_at IS NOT NULL`).Scan(&marked)
	if marked != len(seen) {
		t.Errorf("%d rows marked claimed but %d handed out", marked, len(seen))
	}
	for id, n := range seen {
		if n > 1 {
			t.Errorf("reminder %s claimed %d times", id, n)
		}
	}
}

func TestSendRemindersOnlyDueEvents(t *testing.T) {
	db := testDB(t)
	recipients := fakeResend(t, http.StatusOK)

	insertRemindMe(t, db, insertEvent(t, db, "tomorrow", time.Now().Add(24*time.Hour)), "due")
	insertRemindMe(t, db, insertEvent(t, db, "nextweek", time.Now().Add(7*24*time.Hour)), "early")
	insertRemindMe(t, db, insertEvent(t, db, "inanhour", time.Now().Add(time.Hour)), "late")

	SendReminders(db, "test-key")
	SendReminders(db, "test-key") // a second run must not resend

	got := recipients()
	if len(got) != 1 || got[0] != "due@example.com" {
		t.Errorf("sent to %v, want exactly [due@example.com]", got)
	}
}

func TestSendRemindersStopsAtDailyQuota(t *testing.T) {
	db := testDB(t)
	recipients := fakeResend(t, http.StatusOK)

	// 98 reminders already went out today, for an event outside the window.
	past := insertEvent(t, db, "earlier", time.Now().Add(-time.Hour))
	if _, err := db.Exec(`
		INSERT INTO responses (event_id, name, status, notify_via, reminded_at)
		SELECT $1, 'sent' || g, 'remind_me', 'sent' || g || '@example.com', now()
		FROM generate_series(1, 98) g
	`, past); err != nil {
		t.Fatal(err)
	}

	due := insertEvent(t, db, "tomorrow", time.Now().Add(24*time.Hour))
	for i := range 5 {
		insertRemindMe(t, db, due, fmt.Sprintf("guest%d", i))
	}

	SendReminders(db, "test-key")

	if got := recipients(); len(got) != 2 {
		t.Errorf("sent %d reminders, want 2 (the rest of a %d/day quota)", len(got), resendDailyQuota)
	}
	var unsent int
	db.QueryRow(`SELECT count(*) FROM responses WHERE event_id = $1 AND reminded_at IS NULL`, due).Scan(&unsent)
	if unsent != 3 {
		t.Errorf("%d reminders left unclaimed, want 3", unsent)
	}
}

func TestSendRemindersReleasesFailedBatch(t *testing.T) {
	db := testDB(t)
	fakeResend(t, http.StatusInternalServerError)

	due := insertEvent(t, db, "tomorrow", time.Now().Add(24*time.Hour))
	insertRemindMe(t, db, due, "retry")

	SendReminders(db, "test-key")

	var claimed int
	db.QueryRow(`SELECT count(*) FROM responses WHERE reminded_at IS NOT NULL`).Scan(&claimed)
	if claimed != 0 {
		t.Errorf("%d reminders still marked sent after a failed batch, want 0", claimed)
	}
}

func TestReminderEmailEscapesUserInput(t *testing.T) {
	email := reminderEmail(reminderRow{
		Name:       `<img src=x onerror=alert(1)>`,
		EventTitle: `Party & <b>Games</b>`,
		Location:   `"Court" <4>`,
		EventSlug:  "abc12345",
		EventDate:  time.Date(2026, 9, 20, 18, 0, 0, 0, time.UTC),
	})
	body := email["html"].(string)
	for _, raw := range []string{"<img", "<b>Games", `"Court" <4>`} {
		if strings.Contains(body, raw) {
			t.Errorf("email HTML contains unescaped %q", raw)
		}
	}
}
