package internal

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/lib/pq"
)

type reminderRow struct {
	ResponseID string
	Name       string
	Email      string
	EventTitle string
	EventSlug  string
	EventDate  time.Time
	Location   string
}

// Resend's Free plan allows 100 emails a day (reset at midnight UTC), 10 API
// requests a second, and 100 emails per batch call. Every recipient counts
// toward the daily quota, and reminders are the only mail Ollae sends, so the
// reminders already sent today are the quota used so far.
const (
	resendDailyQuota = 100
	resendBatchSize  = 100
	resendBatchGap   = 250 * time.Millisecond // 4 requests/s, under the 10/s limit
)

var (
	resendClient   = &http.Client{Timeout: 15 * time.Second}
	resendBatchURL = "https://api.resend.com/emails/batch"
)

// SendReminders emails every remind_me response whose event starts in the
// next 23–25 hours. The 2-hour window lets a run that fires slightly late
// still catch its events.
//
// Rows are claimed (reminded_at set) before sending, in one statement with
// SKIP LOCKED, so the in-process loop on each machine and /cron/remind can
// run at the same time without emailing anyone twice. A batch that fails to
// send is released so a later run retries it while the event is still in the
// window.
func SendReminders(db *sql.DB, resendKey string) {
	if resendKey == "" {
		return
	}

	for {
		var sentToday int
		if err := db.QueryRow(`
			SELECT count(*) FROM responses
			WHERE reminded_at >= date_trunc('day', now() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC'
		`).Scan(&sentToday); err != nil {
			log.Printf("reminders: quota query error: %v", err)
			return
		}
		limit := min(resendDailyQuota-sentToday, resendBatchSize)
		if limit <= 0 {
			if due := countDueReminders(db); due > 0 {
				log.Printf("reminders: daily quota of %d reached, %d due reminders not sent", resendDailyQuota, due)
			}
			return
		}

		batch, err := claimReminders(db, limit)
		if err != nil {
			log.Printf("reminders: claim error: %v", err)
			return
		}
		if len(batch) == 0 {
			return
		}

		if err := sendReminderBatch(resendKey, batch); err != nil {
			log.Printf("reminders: batch of %d failed, releasing for retry: %v", len(batch), err)
			releaseReminders(db, batch)
			return
		}
		for _, rem := range batch {
			log.Printf("reminders: sent for event %s (response %s)", rem.EventSlug, rem.ResponseID)
		}

		if len(batch) < limit {
			return
		}
		time.Sleep(resendBatchGap)
	}
}

const dueReminders = `
	FROM responses r
	JOIN events e ON e.id = r.event_id
	WHERE r.status = 'remind_me'
	  AND r.reminded_at IS NULL
	  AND r.notify_via IS NOT NULL
	  AND e.event_date IS NOT NULL
	  AND e.event_date BETWEEN now() + interval '23 hours' AND now() + interval '25 hours'
`

func countDueReminders(db *sql.DB) int {
	var n int
	if err := db.QueryRow(`SELECT count(*) ` + dueReminders).Scan(&n); err != nil {
		log.Printf("reminders: count error: %v", err)
	}
	return n
}

// claimReminders marks up to limit due reminders as sent and returns them.
func claimReminders(db *sql.DB, limit int) ([]reminderRow, error) {
	rows, err := db.Query(`
		WITH due AS (
			SELECT r.id `+dueReminders+`
			ORDER BY e.event_date, r.created_at
			LIMIT $1
			FOR UPDATE OF r SKIP LOCKED
		)
		UPDATE responses r
		SET reminded_at = now()
		FROM due, events e
		WHERE r.id = due.id AND e.id = r.event_id
		RETURNING r.id, r.name, r.notify_via, e.title, e.slug, e.event_date, e.location
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var batch []reminderRow
	for rows.Next() {
		var rem reminderRow
		var location sql.NullString
		if err := rows.Scan(
			&rem.ResponseID, &rem.Name, &rem.Email,
			&rem.EventTitle, &rem.EventSlug, &rem.EventDate, &location,
		); err != nil {
			return nil, err
		}
		rem.Location = location.String
		batch = append(batch, rem)
	}
	return batch, rows.Err()
}

func releaseReminders(db *sql.DB, batch []reminderRow) {
	ids := make([]string, len(batch))
	for i, rem := range batch {
		ids[i] = rem.ResponseID
	}
	if _, err := db.Exec(`UPDATE responses SET reminded_at = NULL WHERE id = ANY($1)`, pq.Array(ids)); err != nil {
		log.Printf("reminders: failed to release %d claimed reminders: %v", len(ids), err)
	}
}

func reminderEmail(rem reminderRow) map[string]any {
	when := rem.EventDate.Format("Monday, January 2 at 3:04 PM")
	where := ""
	if rem.Location != "" {
		where = "\n📍 " + html.EscapeString(rem.Location)
	}

	body := fmt.Sprintf(`
<p>Hey %s 👋</p>
<p>You asked us to remind you about <strong>%s</strong>.</p>
<p>📅 %s%s</p>
<p>Are you in? Tap the link below to RSVP:</p>
<p><a href="https://ollae.app/events/%s">https://ollae.app/events/%s</a></p>
<br>
<p style="color:#94A3B8;font-size:12px;">You received this because you selected "Remind me" on ollae.app. Just ignore this if your plans changed.</p>
`,
		html.EscapeString(rem.Name), html.EscapeString(rem.EventTitle), when, where, rem.EventSlug, rem.EventSlug)

	return map[string]any{
		"from":    "ollae <reminders@ollae.app>",
		"to":      []string{rem.Email},
		"subject": fmt.Sprintf("Reminder: %s is tomorrow", rem.EventTitle),
		"html":    body,
	}
}

func sendReminderBatch(resendKey string, batch []reminderRow) error {
	emails := make([]map[string]any, len(batch))
	for i, rem := range batch {
		emails[i] = reminderEmail(rem)
	}
	payload, _ := json.Marshal(emails)

	req, err := http.NewRequest(http.MethodPost, resendBatchURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+resendKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := resendClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("resend %d: %s", resp.StatusCode, body)
	}
	return nil
}

// StartReminderLoop runs SendReminders every 30 minutes in the background.
// Acts as a best-effort fallback when the machine stays alive; the primary
// trigger is the /cron/remind HTTP endpoint pinged by an external cron service.
func StartReminderLoop(db *sql.DB, resendKey string) {
	go func() {
		SendReminders(db, resendKey)
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			SendReminders(db, resendKey)
		}
	}()
}
