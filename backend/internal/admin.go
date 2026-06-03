package internal

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type AdminEvent struct {
	ID        string     `json:"id"`
	Slug      string     `json:"slug"`
	Title     string     `json:"title"`
	Location  string     `json:"location"`
	EventDate *time.Time `json:"event_date"`
	CreatedAt time.Time  `json:"created_at"`
	Emoji     string     `json:"emoji"`
	Counts    struct {
		In       int `json:"in"`
		Out      int `json:"out"`
		RemindMe int `json:"remind_me"`
	} `json:"counts"`
	Responses []Response `json:"responses"`
}

func adminAuth(secret string, r *http.Request) bool {
	return secret != "" && r.Header.Get("Authorization") == "Bearer "+secret
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
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var totalEvents, totalRSVPs, activeEvents int
	if err := h.DB.QueryRow(`
		SELECT COUNT(*),
		       (SELECT COUNT(*) FROM responses),
		       COUNT(*) FILTER (WHERE event_date > now())
		FROM events
	`).Scan(&totalEvents, &totalRSVPs, &activeEvents); err != nil {
		http.Error(w, "failed to fetch stats", http.StatusInternalServerError)
		return
	}

	retentionMonths := getRetentionMonths(h.DB)

	rows, err := h.DB.Query(`
		SELECT e.id, e.slug, e.title, e.location, e.event_date, e.created_at, e.emoji,
		       COUNT(*) FILTER (WHERE r.status = 'in')        AS count_in,
		       COUNT(*) FILTER (WHERE r.status = 'out')       AS count_out,
		       COUNT(*) FILTER (WHERE r.status = 'remind_me') AS count_remind
		FROM events e
		LEFT JOIN responses r ON r.event_id = e.id
		GROUP BY e.id
		ORDER BY e.created_at DESC
	`)
	if err != nil {
		http.Error(w, "failed to fetch events", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	events := []AdminEvent{}
	for rows.Next() {
		var ev AdminEvent
		if err := rows.Scan(
			&ev.ID, &ev.Slug, &ev.Title, &ev.Location, &ev.EventDate, &ev.CreatedAt, &ev.Emoji,
			&ev.Counts.In, &ev.Counts.Out, &ev.Counts.RemindMe,
		); err != nil {
			http.Error(w, "failed to scan event", http.StatusInternalServerError)
			return
		}

		respRows, err := h.DB.Query(`
			SELECT id, event_id, name, status, guests, notify_via, created_at
			FROM responses WHERE event_id = $1 ORDER BY created_at ASC
		`, ev.ID)
		if err != nil {
			http.Error(w, "failed to fetch responses", http.StatusInternalServerError)
			return
		}
		ev.Responses = []Response{}
		for respRows.Next() {
			var resp Response
			if err := respRows.Scan(&resp.ID, &resp.EventID, &resp.Name, &resp.Status, &resp.Guests, &resp.NotifyVia, &resp.CreatedAt); err != nil {
				respRows.Close()
				http.Error(w, "failed to scan response", http.StatusInternalServerError)
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
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	slug := chi.URLParam(r, "slug")
	if _, err := h.DB.Exec(`DELETE FROM events WHERE slug = $1`, slug); err != nil {
		http.Error(w, "failed to delete event", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *EventHandlers) AdminDeleteResponse(w http.ResponseWriter, r *http.Request) {
	if !adminAuth(h.AdminSecret, r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	if _, err := h.DB.Exec(`DELETE FROM responses WHERE id = $1`, id); err != nil {
		http.Error(w, "failed to delete response", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *EventHandlers) AdminUpdateEvent(w http.ResponseWriter, r *http.Request) {
	if !adminAuth(h.AdminSecret, r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	slug := chi.URLParam(r, "slug")

	var body struct {
		Title     string  `json:"title"`
		Location  string  `json:"location"`
		EventDate *string `json:"event_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if body.Title == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}

	var event Event
	err := h.DB.QueryRow(`
		UPDATE events SET title = $1, location = $2, event_date = $3
		WHERE slug = $4
		RETURNING id, slug, title, location, event_date, created_at, emoji
	`, body.Title, body.Location, body.EventDate, slug).Scan(
		&event.ID, &event.Slug, &event.Title, &event.Location,
		&event.EventDate, &event.CreatedAt, &event.Emoji,
	)
	if err == sql.ErrNoRows {
		http.Error(w, "event not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to update event", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

func (h *EventHandlers) AdminUpdateSettings(w http.ResponseWriter, r *http.Request) {
	if !adminAuth(h.AdminSecret, r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var body struct {
		RetentionMonths int `json:"retention_months"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if body.RetentionMonths < 1 || body.RetentionMonths > 24 {
		http.Error(w, "retention_months must be 1–24", http.StatusBadRequest)
		return
	}

	_, err := h.DB.Exec(`
		INSERT INTO settings (key, value) VALUES ('retention_months', $1)
		ON CONFLICT (key) DO UPDATE SET value = $1
	`, strconv.Itoa(body.RetentionMonths))
	if err != nil {
		http.Error(w, "failed to update settings", http.StatusInternalServerError)
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

func RunCleanup(db *sql.DB) int {
	months := getRetentionMonths(db)
	result, err := db.Exec(`
		DELETE FROM events
		WHERE event_date IS NOT NULL
		  AND event_date < now() - ($1 || ' months')::INTERVAL
	`, strconv.Itoa(months))
	if err != nil {
		return 0
	}
	n, _ := result.RowsAffected()
	return int(n)
}
