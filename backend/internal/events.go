package internal

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

type Event struct {
	ID         string     `json:"id"`
	Slug       string     `json:"slug"`
	Title      string     `json:"title"`
	Location   string     `json:"location"`
	EventDate  *time.Time `json:"event_date"`
	CreatedAt  time.Time  `json:"created_at"`
	Emoji      string     `json:"emoji"`
	AdminToken string     `json:"admin_token,omitempty"` // only returned on create
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
	DB           *sql.DB
	AnthropicKey string
}

func pickEmoji(ctx context.Context, apiKey, title string) string {
	if apiKey == "" {
		return ""
	}
	payload, _ := json.Marshal(map[string]any{
		"model":      "claude-haiku-4-5-20251001",
		"max_tokens": 16,
		"system":     "Reply with a single emoji that best represents the event. Nothing else — one emoji only.",
		"messages":   []map[string]string{{"role": "user", "content": title}},
	})
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return ""
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	resp, err := anthropicClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var apiResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &apiResp); err != nil || len(apiResp.Content) == 0 {
		return ""
	}
	return strings.TrimSpace(apiResp.Content[0].Text)
}

func (h *EventHandlers) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title     string  `json:"title"`
		Location  string  `json:"location"`
		EventDate *string `json:"event_date"`
		Emoji     string  `json:"emoji"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if body.Title == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}

	emoji := body.Emoji
	if emoji == "" {
		emoji = pickEmoji(r.Context(), h.AnthropicKey, body.Title)
	}

	slug, err := gonanoid.Generate("abcdefghijklmnopqrstuvwxyz0123456789", 8)
	if err != nil {
		http.Error(w, "failed to generate slug", http.StatusInternalServerError)
		return
	}
	adminToken, err := gonanoid.Generate("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", 24)
	if err != nil {
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	var event Event
	err = h.DB.QueryRow(`
		INSERT INTO events (slug, title, location, event_date, emoji, admin_token)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, slug, title, location, event_date, created_at, emoji, admin_token
	`, slug, body.Title, body.Location, body.EventDate, emoji, adminToken).Scan(
		&event.ID, &event.Slug, &event.Title, &event.Location,
		&event.EventDate, &event.CreatedAt, &event.Emoji, &event.AdminToken,
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
		SELECT id, slug, title, location, event_date, created_at, emoji
		FROM events WHERE slug = $1
	`, slug).Scan(
		&event.ID, &event.Slug, &event.Title, &event.Location,
		&event.EventDate, &event.CreatedAt, &event.Emoji,
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

	adminToken := r.URL.Query().Get("admin")
	if adminToken == "" {
		http.Error(w, "admin token required", http.StatusUnauthorized)
		return
	}

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
		WHERE slug = $4 AND admin_token = $5
		RETURNING id, slug, title, location, event_date, created_at, emoji
	`, body.Title, body.Location, body.EventDate, slug, adminToken).Scan(
		&event.ID, &event.Slug, &event.Title, &event.Location,
		&event.EventDate, &event.CreatedAt, &event.Emoji,
	)
	if err == sql.ErrNoRows {
		http.Error(w, "event not found or invalid token", http.StatusForbidden)
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
