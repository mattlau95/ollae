package internal

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

type Event struct {
	ID        string     `json:"id"`
	Slug      string     `json:"slug"`
	Title     string     `json:"title"`
	Location  string     `json:"location"`
	EventDate *time.Time `json:"event_date"`
	CreatedAt time.Time  `json:"created_at"`
}

type Response struct {
	ID        string    `json:"id"`
	EventID   string    `json:"event_id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	NotifyVia *string   `json:"notify_via"`
	CreatedAt time.Time `json:"created_at"`
}

type EventHandlers struct {
	DB *sql.DB
}

func (h *EventHandlers) CreateEvent(w http.ResponseWriter, r *http.Request) {
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

	slug, err := gonanoid.Generate("abcdefghijklmnopqrstuvwxyz0123456789", 8)
	if err != nil {
		http.Error(w, "failed to generate slug", http.StatusInternalServerError)
		return
	}

	var event Event
	err = h.DB.QueryRow(`
		INSERT INTO events (slug, title, location, event_date)
		VALUES ($1, $2, $3, $4)
		RETURNING id, slug, title, location, event_date, created_at
	`, slug, body.Title, body.Location, body.EventDate).Scan(
		&event.ID, &event.Slug, &event.Title, &event.Location,
		&event.EventDate, &event.CreatedAt,
	)
	if err != nil {
		http.Error(w, "failed to create event", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(event)
}

func (h *EventHandlers) GetEvent(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	var event Event
	err := h.DB.QueryRow(`
		SELECT id, slug, title, location, event_date, created_at
		FROM events WHERE slug = $1
	`, slug).Scan(
		&event.ID, &event.Slug, &event.Title, &event.Location,
		&event.EventDate, &event.CreatedAt,
	)
	if err == sql.ErrNoRows {
		http.Error(w, "event not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to fetch event", http.StatusInternalServerError)
		return
	}

	rows, err := h.DB.Query(`
		SELECT id, event_id, name, status, notify_via, created_at
		FROM responses WHERE event_id = $1
		ORDER BY created_at ASC
	`, event.ID)
	if err != nil {
		http.Error(w, "failed to fetch responses", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	responses := []Response{}
	for rows.Next() {
		var resp Response
		if err := rows.Scan(&resp.ID, &resp.EventID, &resp.Name,
			&resp.Status, &resp.NotifyVia, &resp.CreatedAt); err != nil {
			http.Error(w, "failed to scan response", http.StatusInternalServerError)
			return
		}
		responses = append(responses, resp)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"event":     event,
		"responses": responses,
	})
}

func (h *EventHandlers) UpdateEvent(w http.ResponseWriter, r *http.Request) {
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
		UPDATE events
		SET title = $1, location = $2, event_date = $3
		WHERE slug = $4
		RETURNING id, slug, title, location, event_date, created_at
	`, body.Title, body.Location, body.EventDate, slug).Scan(
		&event.ID, &event.Slug, &event.Title, &event.Location,
		&event.EventDate, &event.CreatedAt,
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

func (h *EventHandlers) SubmitRSVP(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	var body struct {
		Name      string  `json:"name"`
		Status    string  `json:"status"`
		NotifyVia *string `json:"notify_via"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if body.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	validStatuses := map[string]bool{"in": true, "out": true, "remind_me": true}
	if !validStatuses[body.Status] {
		http.Error(w, "status must be in, out, or remind_me", http.StatusBadRequest)
		return
	}
	if body.Status == "remind_me" && (body.NotifyVia == nil || *body.NotifyVia == "") {
		http.Error(w, "notify_via is required when status is remind_me", http.StatusBadRequest)
		return
	}

	// Look up the event by slug
	var eventID string
	err := h.DB.QueryRow(`SELECT id FROM events WHERE slug = $1`, slug).Scan(&eventID)
	if err == sql.ErrNoRows {
		http.Error(w, "event not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to fetch event", http.StatusInternalServerError)
		return
	}

	// Upsert — insert or update if name already exists for this event
	_, err = h.DB.Exec(`
		INSERT INTO responses (event_id, name, status, notify_via)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (event_id, lower(name))
		DO UPDATE SET status = EXCLUDED.status, notify_via = EXCLUDED.notify_via
	`, eventID, body.Name, body.Status, body.NotifyVia)
	if err != nil {
		log.Printf("upsert error: %v", err)
		http.Error(w, "failed to save response", http.StatusInternalServerError)
		return
	}

	// Return the full updated response list
	rows, err := h.DB.Query(`
		SELECT id, event_id, name, status, notify_via, created_at
		FROM responses WHERE event_id = $1
		ORDER BY created_at ASC
	`, eventID)
	if err != nil {
		http.Error(w, "failed to fetch responses", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	responses := []Response{}
	for rows.Next() {
		var resp Response
		if err := rows.Scan(&resp.ID, &resp.EventID, &resp.Name,
			&resp.Status, &resp.NotifyVia, &resp.CreatedAt); err != nil {
			http.Error(w, "failed to scan response", http.StatusInternalServerError)
			return
		}
		responses = append(responses, resp)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}
