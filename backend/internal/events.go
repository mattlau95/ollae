package internal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
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
	IsDemo     bool       `json:"is_demo"`
	AppendOnly bool       `json:"append_only"`
	// RemindersOff keeps "Remind me" as an answer but never takes an email,
	// for events like the portfolio guestbook that nobody will attend.
	RemindersOff bool   `json:"reminders_off"`
	AdminToken   string `json:"admin_token,omitempty"` // only returned on create
}

// ogWarmBase is where CreateEvent pre-warms the OG image; tests clear it.
var ogWarmBase = "https://ollae.app/og/"

// eventColumns and scanEvent read an Event the same way in every query.
const eventColumns = `id, slug, title, COALESCE(location, ''), event_date, created_at, emoji, is_demo, append_only, reminders_off`

func scanEvent(row interface{ Scan(...any) error }, e *Event, extra ...any) error {
	return row.Scan(append([]any{
		&e.ID, &e.Slug, &e.Title, &e.Location, &e.EventDate, &e.CreatedAt, &e.Emoji, &e.IsDemo, &e.AppendOnly, &e.RemindersOff,
	}, extra...)...)
}

type Response struct {
	ID        string    `json:"id"`
	EventID   string    `json:"event_id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Guests    int       `json:"guests"`
	NotifyVia *string   `json:"notify_via"`
	CreatedAt time.Time `json:"created_at"`
}

// PublicResponse is a Response as anyone with the event link sees it. It has
// no NotifyVia: reminder emails are visible only to /admin.
type PublicResponse struct {
	ID        string    `json:"id"`
	EventID   string    `json:"event_id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Guests    int       `json:"guests"`
	CreatedAt time.Time `json:"created_at"`
}

// publicResponses lists an event's responses, oldest first, without emails.
func publicResponses(db *sql.DB, eventID string) ([]PublicResponse, error) {
	rows, err := db.Query(`
		SELECT id, event_id, name, status, guests, created_at
		FROM responses WHERE event_id = $1
		ORDER BY created_at ASC
	`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	responses := []PublicResponse{}
	for rows.Next() {
		var resp PublicResponse
		if err := rows.Scan(&resp.ID, &resp.EventID, &resp.Name,
			&resp.Status, &resp.Guests, &resp.CreatedAt); err != nil {
			return nil, err
		}
		responses = append(responses, resp)
	}
	return responses, rows.Err()
}

type EventHandlers struct {
	DB             *sql.DB
	AnthropicKey   string
	AdminSecret    string
	FBAppToken     string
	ClaudeDailyCap int
	Blocklist      *Blocklist // nil lets everything through
}

func pickEmoji(ctx context.Context, h *EventHandlers, title string) string {
	if h.AnthropicKey == "" || takeClaudeCall(ctx, h.DB, h.ClaudeDailyCap) != nil {
		return ""
	}
	payload, _ := json.Marshal(map[string]any{
		"model":      "claude-haiku-4-5-20251001",
		"max_tokens": 16,
		"system":     "Reply with a single emoji that best represents the event. Nothing else — one emoji only.",
		"messages":   []map[string]string{{"role": "user", "content": title}},
	})
	// The emoji is a nicety; don't hold up event creation for long.
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	status, raw, err := callAnthropic(ctx, h.AnthropicKey, payload)
	if err != nil || status != http.StatusOK {
		return ""
	}
	var apiResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &apiResp); err != nil || len(apiResp.Content) == 0 {
		return ""
	}
	if emoji := strings.TrimSpace(apiResp.Content[0].Text); validEmoji(emoji) {
		return emoji
	}
	return ""
}

// eventFields validates the fields shared by create and update.
func eventFields(title, location string, eventDate *string, blocklist *Blocklist) (string, string, error) {
	title, err := cleanField(title, maxTitleLen, "Event names", blocklist)
	if err != nil {
		return "", "", err
	}
	if title == "" {
		return "", "", validationError{"Give your event a name."}
	}
	location, err = cleanField(location, maxLocationLen, "Locations", blocklist)
	if err != nil {
		return "", "", err
	}
	if eventDate != nil && !validEventDate(*eventDate) {
		return "", "", validationError{"That date or time doesn't look right."}
	}
	return title, location, nil
}

// writeValidation sends a validationError's message, or a generic 400.
func writeValidation(w http.ResponseWriter, err error) {
	var ve validationError
	if errors.As(err, &ve) {
		JSONErrorCode(w, http.StatusBadRequest, "invalid", ve.msg)
		return
	}
	JSONError(w, http.StatusBadRequest, "invalid request body")
}

func (h *EventHandlers) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title     string  `json:"title"`
		Location  string  `json:"location"`
		EventDate *string `json:"event_date"`
		Emoji     string  `json:"emoji"`
		Demo      bool    `json:"demo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	title, location, err := eventFields(body.Title, body.Location, body.EventDate, h.Blocklist)
	if err != nil {
		writeValidation(w, err)
		return
	}

	emoji := body.Emoji
	if !validEmoji(emoji) {
		emoji = pickEmoji(r.Context(), h, title)
	}

	slug, err := gonanoid.Generate("abcdefghijklmnopqrstuvwxyz0123456789", 8)
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "failed to generate slug")
		return
	}
	adminToken, err := gonanoid.Generate("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", 24)
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	var event Event
	err = scanEvent(h.DB.QueryRow(`
		INSERT INTO events (slug, title, location, event_date, emoji, admin_token, is_demo)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING `+eventColumns+`, admin_token
	`, slug, title, location, body.EventDate, emoji, adminToken, body.Demo), &event, &event.AdminToken)
	if err != nil {
		log.Printf("create event: %v", err)
		JSONError(w, http.StatusInternalServerError, "failed to create event")
		return
	}

	// Pre-warm the OG image so the CDN has it cached before Facebook's scraper
	// hits the URL — FB's first scrape wins and is cached for ~30 days.
	if ogWarmBase != "" {
		warmCtx, warmCancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer warmCancel()
		if warmReq, err := http.NewRequestWithContext(warmCtx, http.MethodGet, ogWarmBase+slug+"?v=2", nil); err == nil {
			if warmResp, err := http.DefaultClient.Do(warmReq); err == nil {
				warmResp.Body.Close()
			} else {
				log.Printf("OG warm failed %s: %v", slug, err)
			}
		}
	}

	// Demo events live for 3 days and aren't shared into group chats.
	if !event.IsDemo {
		go h.pingFBScraper(slug)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(event)
}

// pingFBScraper tells Facebook to re-scrape the event URL. Messenger group chats
// maintain a per-conversation preview cache that won't update from the CDN cache
// alone — this forces Facebook to refresh their graph cache for the URL.
func (h *EventHandlers) pingFBScraper(slug string) {
	if h.FBAppToken == "" {
		log.Printf("FB scrape ping skipped %s: FB_APP_TOKEN not set", slug)
		return
	}
	params := url.Values{
		"id":           {"https://ollae.app/events/" + slug},
		"scrape":       {"true"},
		"access_token": {h.FBAppToken},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://graph.facebook.com/?"+params.Encode(), nil)
	if err != nil {
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("FB scrape ping failed %s: %v", slug, err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	if resp.StatusCode != http.StatusOK {
		log.Printf("FB scrape ping error %s: status=%d body=%s", slug, resp.StatusCode, body)
	} else {
		log.Printf("FB scrape ping ok %s: %s", slug, body)
	}
}

// RescrapeEvent lets an admin manually re-trigger the FB scraper ping for an
// existing event — useful when a group chat has a stale cached preview.
func (h *EventHandlers) RescrapeEvent(w http.ResponseWriter, r *http.Request) {
	if !adminAuth(h.AdminSecret, r) {
		JSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	slug := chi.URLParam(r, "slug")
	go h.pingFBScraper(slug)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func (h *EventHandlers) GetEvent(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	var event Event
	err := scanEvent(h.DB.QueryRow(`SELECT `+eventColumns+` FROM events WHERE slug = $1`, slug), &event)
	if err == sql.ErrNoRows {
		JSONError(w, http.StatusNotFound, "event not found")
		return
	}
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "failed to fetch event")
		return
	}

	responses, err := publicResponses(h.DB, event.ID)
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "failed to fetch responses")
		return
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
		JSONError(w, http.StatusUnauthorized, "admin token required")
		return
	}

	var body struct {
		Title     string  `json:"title"`
		Location  string  `json:"location"`
		EventDate *string `json:"event_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	title, location, err := eventFields(body.Title, body.Location, body.EventDate, h.Blocklist)
	if err != nil {
		writeValidation(w, err)
		return
	}

	var event Event
	err = scanEvent(h.DB.QueryRow(`
		UPDATE events
		SET title = $1, location = $2, event_date = $3
		WHERE slug = $4 AND admin_token = $5
		RETURNING `+eventColumns+`
	`, title, location, body.EventDate, slug, adminToken), &event)
	if err == sql.ErrNoRows {
		JSONError(w, http.StatusForbidden, "event not found or invalid token")
		return
	}
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "failed to update event")
		return
	}

	if !event.IsDemo {
		go h.pingFBScraper(slug)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

func (h *EventHandlers) SubmitRSVP(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	var body struct {
		Name      string  `json:"name"`
		Status    string  `json:"status"`
		Guests    int     `json:"guests"`
		NotifyVia *string `json:"notify_via"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	name, err := cleanField(body.Name, maxNameLen, "Names", h.Blocklist)
	if err != nil {
		writeValidation(w, err)
		return
	}
	if name == "" {
		JSONErrorCode(w, http.StatusBadRequest, "invalid", "Please enter your name.")
		return
	}
	validStatuses := map[string]bool{"in": true, "out": true, "remind_me": true}
	if !validStatuses[body.Status] {
		JSONError(w, http.StatusBadRequest, "status must be in, out, or remind_me")
		return
	}
	if body.Guests < 0 || body.Guests > maxGuests {
		JSONErrorCode(w, http.StatusBadRequest, "invalid", fmt.Sprintf("You can bring up to %d guests.", maxGuests))
		return
	}
	// guests only makes sense for "in" RSVPs
	if body.Status != "in" {
		body.Guests = 0
	}

	var eventID string
	var isDemo, appendOnly, remindersOff bool
	err = h.DB.QueryRow(`SELECT id, is_demo, append_only, reminders_off FROM events WHERE slug = $1`, slug).
		Scan(&eventID, &isDemo, &appendOnly, &remindersOff)
	if err == sql.ErrNoRows {
		JSONError(w, http.StatusNotFound, "event not found")
		return
	}
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "failed to fetch event")
		return
	}

	email := ""
	if body.NotifyVia != nil {
		email = strings.TrimSpace(*body.NotifyVia)
	}
	switch {
	case isDemo && body.Status == "remind_me":
		JSONErrorCode(w, http.StatusBadRequest, "invalid", "Reminders are turned off for demo events.")
		return
	case remindersOff && email != "":
		// "Remind me" is still a valid answer here; it just never takes an email.
		JSONErrorCode(w, http.StatusBadRequest, "invalid", "This event doesn't send reminders, so it doesn't take an email.")
		return
	case remindersOff:
		body.NotifyVia = nil
	case body.Status == "remind_me":
		if !validEmail(email) {
			JSONErrorCode(w, http.StatusBadRequest, "invalid", "That email address doesn't look right.")
			return
		}
		body.NotifyVia = &email
	default:
		body.NotifyVia = nil
	}

	// On an append-only event a name already on the list is kept as it is,
	// so nobody can change someone else's answer. Otherwise resubmitting a
	// name updates that RSVP.
	onConflict := `DO UPDATE SET status = EXCLUDED.status, guests = EXCLUDED.guests, notify_via = EXCLUDED.notify_via`
	if appendOnly {
		onConflict = `DO NOTHING`
	}
	result, err := h.DB.Exec(`
		INSERT INTO responses (event_id, name, status, guests, notify_via)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (event_id, lower(name)) `+onConflict,
		eventID, name, body.Status, body.Guests, body.NotifyVia)
	if err != nil {
		log.Printf("upsert error: %v", err)
		JSONError(w, http.StatusInternalServerError, "failed to save response")
		return
	}
	if n, _ := result.RowsAffected(); n == 0 {
		first := strings.Fields(name)[0]
		JSONErrorCode(w, http.StatusConflict, "name_taken", fmt.Sprintf(
			"Someone named %s is already on the list. Add a last initial, like “%s K.”", name, first))
		return
	}

	// Return the full updated response list
	responses, err := publicResponses(h.DB, eventID)
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "failed to fetch responses")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}

// DeleteResponse lets the organizer remove one RSVP with the edit link's
// token, and returns the updated list.
func (h *EventHandlers) DeleteResponse(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	id := chi.URLParam(r, "id")
	adminToken := r.URL.Query().Get("admin")
	if adminToken == "" {
		JSONError(w, http.StatusUnauthorized, "admin token required")
		return
	}

	var eventID string
	err := h.DB.QueryRow(`
		SELECT id FROM events WHERE slug = $1 AND admin_token = $2
	`, slug, adminToken).Scan(&eventID)
	if err == sql.ErrNoRows {
		JSONError(w, http.StatusForbidden, "event not found or invalid token")
		return
	}
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "failed to fetch event")
		return
	}

	// id is compared as text so a malformed id is simply not found.
	if _, err := h.DB.Exec(`DELETE FROM responses WHERE id::text = $1 AND event_id = $2`, id, eventID); err != nil {
		JSONError(w, http.StatusInternalServerError, "failed to delete response")
		return
	}

	responses, err := publicResponses(h.DB, eventID)
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "failed to fetch responses")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}
