package internal

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
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

// SendReminders finds all remind_me responses whose events start within the next
// 25 hours, have not been reminded yet, and sends a reminder email via Resend.
// The 25-hour window (vs exactly 24h) provides a buffer so a run that fires
// slightly late still catches events in the target window.
func SendReminders(db *sql.DB, resendKey string) {
	if resendKey == "" {
		return
	}

	rows, err := db.Query(`
		SELECT r.id, r.name, r.notify_via, e.title, e.slug, e.event_date, e.location
		FROM responses r
		JOIN events e ON e.id = r.event_id
		WHERE r.status = 'remind_me'
		  AND r.reminded_at IS NULL
		  AND r.notify_via IS NOT NULL
		  AND e.event_date IS NOT NULL
		  AND e.event_date BETWEEN now() + interval '23 hours' AND now() + interval '25 hours'
	`)
	if err != nil {
		log.Printf("reminders: query error: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var rem reminderRow
		var location sql.NullString
		if err := rows.Scan(
			&rem.ResponseID, &rem.Name, &rem.Email,
			&rem.EventTitle, &rem.EventSlug, &rem.EventDate, &location,
		); err != nil {
			log.Printf("reminders: scan error: %v", err)
			continue
		}
		if location.Valid {
			rem.Location = location.String
		}

		if err := sendReminderEmail(resendKey, rem); err != nil {
			log.Printf("reminders: failed to send to %s: %v", rem.Email, err)
			continue
		}

		_, err = db.Exec(`UPDATE responses SET reminded_at = now() WHERE id = $1`, rem.ResponseID)
		if err != nil {
			log.Printf("reminders: failed to mark reminded for %s: %v", rem.ResponseID, err)
		} else {
			log.Printf("reminders: sent to %s for event %s", rem.Email, rem.EventSlug)
		}
	}
}

func sendReminderEmail(resendKey string, rem reminderRow) error {
	subject := fmt.Sprintf("Reminder: %s is tomorrow", rem.EventTitle)

	when := rem.EventDate.Format("Monday, January 2 at 3:04 PM")
	where := ""
	if rem.Location != "" {
		where = fmt.Sprintf("\n📍 %s", rem.Location)
	}

	html := fmt.Sprintf(`
<p>Hey %s 👋</p>
<p>You asked us to remind you about <strong>%s</strong>.</p>
<p>📅 %s%s</p>
<p>Are you in? Tap the link below to RSVP:</p>
<p><a href="https://ollae.app/events/%s">https://ollae.app/events/%s</a></p>
<br>
<p style="color:#94A3B8;font-size:12px;">You received this because you selected "Remind me" on ollae.app. Just ignore this if your plans changed.</p>
`,
		rem.Name, rem.EventTitle, when, where, rem.EventSlug, rem.EventSlug)

	payload, _ := json.Marshal(map[string]any{
		"from":    "ollae <reminders@ollae.app>",
		"to":      []string{rem.Email},
		"subject": subject,
		"html":    html,
	})

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+resendKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// StartReminderLoop runs SendReminders every 30 minutes in the background.
func StartReminderLoop(db *sql.DB, resendKey string) {
	go func() {
		// Run once at startup (catches anything missed while the machine was stopped)
		SendReminders(db, resendKey)
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			SendReminders(db, resendKey)
		}
	}()
}
