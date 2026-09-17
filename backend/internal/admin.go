package internal

import (
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type AdminEvent struct {
	ID           string     `json:"id"`
	Slug         string     `json:"slug"`
	Title        string     `json:"title"`
	Location     string     `json:"location"`
	EventDate    *time.Time `json:"event_date"`
	CreatedAt    time.Time  `json:"created_at"`
	Emoji        string     `json:"emoji"`
	IsDemo       bool       `json:"is_demo"`
	AppendOnly   bool       `json:"append_only"`
	RemindersOff bool       `json:"reminders_off"`
	Counts       struct {
		In       int `json:"in"`
		Out      int `json:"out"`
		RemindMe int `json:"remind_me"`
	} `json:"counts"`
	Responses []Response `json:"responses"`
}

func adminAuth(secret string, r *http.Request) bool {
	if secret == "" {
		return false
	}
	got := []byte(r.Header.Get("Authorization"))
	want := []byte("Bearer " + secret)
	return subtle.ConstantTimeCompare(got, want) == 1
}

func getRetentionMonths(db *sql.DB) int {
	var val string
	if err := db.QueryRow(`SELECT value FROM settings WHERE key = 'retention_months'`).Scan(&val); err == nil {
		if v, err := strconv.Atoi(val); err == nil && v > 0 {
			return v
		}
	}
	return 2
}

func (h *EventHandlers) AdminGetEvents(w http.ResponseWriter, r *http.Request) {
	if !adminAuth(h.AdminSecret, r) {
		JSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var totalEvents, totalRSVPs, activeEvents int
	if err := h.DB.QueryRow(`
		SELECT COUNT(*),
		       (SELECT COUNT(*) FROM responses),
		       COUNT(*) FILTER (WHERE event_date > now())
		FROM events
	`).Scan(&totalEvents, &totalRSVPs, &activeEvents); err != nil {
		JSONError(w, http.StatusInternalServerError, "failed to fetch stats")
		return
	}

	retentionMonths := getRetentionMonths(h.DB)

	rows, err := h.DB.Query(`
		SELECT e.id, e.slug, e.title, e.location, e.event_date, e.created_at, e.emoji,
		       e.is_demo, e.append_only, e.reminders_off,
		       COUNT(*) FILTER (WHERE r.status = 'in')        AS count_in,
		       COUNT(*) FILTER (WHERE r.status = 'out')       AS count_out,
		       COUNT(*) FILTER (WHERE r.status = 'remind_me') AS count_remind
		FROM events e
		LEFT JOIN responses r ON r.event_id = e.id
		GROUP BY e.id
		ORDER BY e.created_at DESC
	`)
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "failed to fetch events")
		return
	}
	defer rows.Close()

	events := []AdminEvent{}
	for rows.Next() {
		var ev AdminEvent
		if err := rows.Scan(
			&ev.ID, &ev.Slug, &ev.Title, &ev.Location, &ev.EventDate, &ev.CreatedAt, &ev.Emoji,
			&ev.IsDemo, &ev.AppendOnly, &ev.RemindersOff,
			&ev.Counts.In, &ev.Counts.Out, &ev.Counts.RemindMe,
		); err != nil {
			JSONError(w, http.StatusInternalServerError, "failed to scan event")
			return
		}

		respRows, err := h.DB.Query(`
			SELECT id, event_id, name, status, guests, notify_via, created_at
			FROM responses WHERE event_id = $1 ORDER BY created_at ASC
		`, ev.ID)
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "failed to fetch responses")
			return
		}
		ev.Responses = []Response{}
		for respRows.Next() {
			var resp Response
			if err := respRows.Scan(&resp.ID, &resp.EventID, &resp.Name, &resp.Status, &resp.Guests, &resp.NotifyVia, &resp.CreatedAt); err != nil {
				respRows.Close()
				JSONError(w, http.StatusInternalServerError, "failed to scan response")
				return
			}
			ev.Responses = append(ev.Responses, resp)
		}
		respRows.Close()

		events = append(events, ev)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"stats": map[string]int{
			"total_events":  totalEvents,
			"total_rsvps":   totalRSVPs,
			"active_events": activeEvents,
		},
		"retention_months": retentionMonths,
		"events":           events,
	})
}

func (h *EventHandlers) AdminDeleteEvent(w http.ResponseWriter, r *http.Request) {
	if !adminAuth(h.AdminSecret, r) {
		JSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	slug := chi.URLParam(r, "slug")
	if _, err := h.DB.Exec(`DELETE FROM events WHERE slug = $1`, slug); err != nil {
		JSONError(w, http.StatusInternalServerError, "failed to delete event")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *EventHandlers) AdminDeleteResponse(w http.ResponseWriter, r *http.Request) {
	if !adminAuth(h.AdminSecret, r) {
		JSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	if _, err := h.DB.Exec(`DELETE FROM responses WHERE id = $1`, id); err != nil {
		JSONError(w, http.StatusInternalServerError, "failed to delete response")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *EventHandlers) AdminUpdateEvent(w http.ResponseWriter, r *http.Request) {
	if !adminAuth(h.AdminSecret, r) {
		JSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	slug := chi.URLParam(r, "slug")

	var body struct {
		Title     string  `json:"title"`
		Location  string  `json:"location"`
		EventDate *string `json:"event_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Title == "" {
		JSONError(w, http.StatusBadRequest, "title is required")
		return
	}

	var event Event
	err := scanEvent(h.DB.QueryRow(`
		UPDATE events SET title = $1, location = $2, event_date = $3
		WHERE slug = $4
		RETURNING `+eventColumns+`
	`, body.Title, body.Location, body.EventDate, slug), &event)
	if err == sql.ErrNoRows {
		JSONError(w, http.StatusNotFound, "event not found")
		return
	}
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "failed to update event")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

// AdminSetAppendOnly turns an event's append-only setting on or off.
func (h *EventHandlers) AdminSetAppendOnly(w http.ResponseWriter, r *http.Request) {
	if !adminAuth(h.AdminSecret, r) {
		JSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		AppendOnly bool `json:"append_only"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := h.DB.Exec(`UPDATE events SET append_only = $1 WHERE slug = $2`, body.AppendOnly, chi.URLParam(r, "slug"))
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "failed to update event")
		return
	}
	if n, _ := result.RowsAffected(); n == 0 {
		JSONError(w, http.StatusNotFound, "event not found")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"append_only": body.AppendOnly})
}

// AdminSetRemindersOff turns an event's reminders off or on. Turning them
// off also clears any reminder emails already stored for the event, since
// none will ever be sent.
func (h *EventHandlers) AdminSetRemindersOff(w http.ResponseWriter, r *http.Request) {
	if !adminAuth(h.AdminSecret, r) {
		JSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		RemindersOff bool `json:"reminders_off"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	var eventID string
	err := h.DB.QueryRow(`UPDATE events SET reminders_off = $1 WHERE slug = $2 RETURNING id`,
		body.RemindersOff, chi.URLParam(r, "slug")).Scan(&eventID)
	if err == sql.ErrNoRows {
		JSONError(w, http.StatusNotFound, "event not found")
		return
	}
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "failed to update event")
		return
	}
	cleared := 0
	if body.RemindersOff {
		result, err := h.DB.Exec(`UPDATE responses SET notify_via = NULL WHERE event_id = $1 AND notify_via IS NOT NULL`, eventID)
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "failed to clear reminder emails")
			return
		}
		n, _ := result.RowsAffected()
		cleared = int(n)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"reminders_off": body.RemindersOff, "emails_cleared": cleared})
}

const maxBlockedTermLen = 60

// AdminListBlockedTerms returns every blocked term, alphabetically.
func (h *EventHandlers) AdminListBlockedTerms(w http.ResponseWriter, r *http.Request) {
	if !adminAuth(h.AdminSecret, r) {
		JSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	rows, err := h.DB.Query(`SELECT term FROM blocked_terms ORDER BY term`)
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "failed to fetch blocked terms")
		return
	}
	defer rows.Close()
	terms := []string{}
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			JSONError(w, http.StatusInternalServerError, "failed to fetch blocked terms")
			return
		}
		terms = append(terms, t)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]string{"terms": terms})
}

// AdminAddBlockedTerm and AdminRemoveBlockedTerm take the term in a JSON
// body, never the URL, so it doesn't end up in request logs.
func (h *EventHandlers) AdminAddBlockedTerm(w http.ResponseWriter, r *http.Request) {
	h.changeBlockedTerm(w, r, `INSERT INTO blocked_terms (term) VALUES ($1) ON CONFLICT (term) DO NOTHING`)
}

func (h *EventHandlers) AdminRemoveBlockedTerm(w http.ResponseWriter, r *http.Request) {
	h.changeBlockedTerm(w, r, `DELETE FROM blocked_terms WHERE term = $1`)
}

func (h *EventHandlers) changeBlockedTerm(w http.ResponseWriter, r *http.Request, query string) {
	if !adminAuth(h.AdminSecret, r) {
		JSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		Term string `json:"term"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	term := NormalizeTerm(body.Term)
	if term == "" {
		JSONError(w, http.StatusBadRequest, "Enter a word or phrase.")
		return
	}
	if len([]rune(term)) > maxBlockedTermLen {
		JSONError(w, http.StatusBadRequest, "Terms can be up to 60 characters.")
		return
	}

	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "failed to update blocked terms")
		return
	}
	defer tx.Rollback()
	if _, err := tx.Exec(query, term); err != nil {
		JSONError(w, http.StatusInternalServerError, "failed to update blocked terms")
		return
	}
	// Every machine polls this version and reloads when it changes.
	if _, err := tx.Exec(`
		INSERT INTO settings (key, value) VALUES ('blocked_terms_version', $1)
		ON CONFLICT (key) DO UPDATE SET value = $1
	`, strconv.FormatInt(time.Now().UnixNano(), 10)); err != nil {
		JSONError(w, http.StatusInternalServerError, "failed to update blocked terms")
		return
	}
	if err := tx.Commit(); err != nil {
		JSONError(w, http.StatusInternalServerError, "failed to update blocked terms")
		return
	}
	if h.Blocklist != nil {
		if err := h.Blocklist.Refresh(r.Context()); err != nil {
			log.Printf("blocklist: refresh after change failed: %v", err)
		}
	}
	h.AdminListBlockedTerms(w, r)
}

func (h *EventHandlers) AdminUpdateSettings(w http.ResponseWriter, r *http.Request) {
	if !adminAuth(h.AdminSecret, r) {
		JSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body struct {
		RetentionMonths int `json:"retention_months"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.RetentionMonths < 1 || body.RetentionMonths > 24 {
		JSONError(w, http.StatusBadRequest, "retention_months must be 1–24")
		return
	}

	_, err := h.DB.Exec(`
		INSERT INTO settings (key, value) VALUES ('retention_months', $1)
		ON CONFLICT (key) DO UPDATE SET value = $1
	`, strconv.Itoa(body.RetentionMonths))
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "failed to update settings")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"retention_months": body.RetentionMonths})
}

func (h *EventHandlers) GetRetention(w http.ResponseWriter, r *http.Request) {
	months := getRetentionMonths(h.DB)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"retention_months": months})
}

// CleanupResult counts what one RunCleanup pass removed.
type CleanupResult struct {
	EventsDeleted     int `json:"deleted"`
	DemoEventsDeleted int `json:"demo_deleted"`
	EmailsCleared     int `json:"emails_cleared"`
}

// demoLifetime is how long an event created from the portfolio embed lives.
const demoLifetime = "3 days"

// RunCleanup deletes events past the retention period and demo events older
// than demoLifetime, and clears reminder emails once their event is over.
// The email is collected for one reminder, so it has no use after the event.
// Event dates are stored as the organizer's wall-clock time labeled UTC, so
// "over" waits an extra day to cover any timezone.
func RunCleanup(db *sql.DB) CleanupResult {
	var res CleanupResult
	exec := func(what string, n *int, query string, args ...any) {
		result, err := db.Exec(query, args...)
		if err != nil {
			log.Printf("cleanup: %s: %v", what, err)
			return
		}
		rows, _ := result.RowsAffected()
		*n = int(rows)
	}

	exec("delete expired events", &res.EventsDeleted, `
		DELETE FROM events
		WHERE event_date IS NOT NULL
		  AND event_date < now() - ($1 || ' months')::INTERVAL
	`, strconv.Itoa(getRetentionMonths(db)))

	exec("delete demo events", &res.DemoEventsDeleted, `
		DELETE FROM events
		WHERE is_demo AND created_at < now() - $1::INTERVAL
	`, demoLifetime)

	exec("clear emails", &res.EmailsCleared, `
		UPDATE responses r
		SET notify_via = NULL
		FROM events e
		WHERE e.id = r.event_id
		  AND r.notify_via IS NOT NULL
		  AND e.event_date IS NOT NULL
		  AND e.event_date < now() - interval '1 day'
	`)
	return res
}

// StartCleanupLoop runs RunCleanup hourly, so demo events go close to their
// 3 days rather than waiting for the daily /cron/cleanup call.
func StartCleanupLoop(db *sql.DB) {
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if res := RunCleanup(db); res != (CleanupResult{}) {
				log.Printf("cleanup: %+v", res)
			}
		}
	}()
}
