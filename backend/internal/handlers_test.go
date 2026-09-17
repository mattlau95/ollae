package internal

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

// testServer routes the public handlers the way cmd/main.go does.
func testServer(t *testing.T, h *EventHandlers) *httptest.Server {
	t.Helper()
	r := chi.NewRouter()
	r.Use(NoFraming)
	r.Post("/events", h.CreateEvent)
	r.Get("/events/{slug}", h.GetEvent)
	r.Patch("/events/{slug}", h.UpdateEvent)
	r.Post("/events/{slug}/rsvp", h.SubmitRSVP)
	r.Delete("/events/{slug}/responses/{id}", h.DeleteResponse)
	r.Get("/og-preview/{slug}", h.OGPreview)
	r.Post("/parse-event", h.ParseEvent)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

type apiError struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// noRedirects stops tests from following a redirect out to production.
var noRedirects = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

func do(t *testing.T, method, url, body string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := noRedirects.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b
}

func decodeError(t *testing.T, b []byte) apiError {
	t.Helper()
	var e apiError
	if err := json.Unmarshal(b, &e); err != nil {
		t.Fatalf("not a JSON error: %s", b)
	}
	return e
}

func TestRSVPAppendOnly(t *testing.T) {
	db := testDB(t)
	srv := testServer(t, &EventHandlers{DB: db})
	insertEvent(t, db, "guestbook", time.Now().Add(24*time.Hour))
	insertEvent(t, db, "normal", time.Now().Add(24*time.Hour))
	db.Exec(`UPDATE events SET append_only = true WHERE slug = 'guestbook'`)

	if code, b := do(t, "POST", srv.URL+"/events/guestbook/rsvp", `{"name":"Alex","status":"in"}`); code != 200 {
		t.Fatalf("first RSVP: %d %s", code, b)
	}
	code, b := do(t, "POST", srv.URL+"/events/guestbook/rsvp", `{"name":"  alex ","status":"out"}`)
	if code != http.StatusConflict {
		t.Fatalf("duplicate name on append-only event: got %d %s, want 409", code, b)
	}
	e := decodeError(t, b)
	if e.Code != "name_taken" || !strings.Contains(e.Error, "last initial") {
		t.Errorf("unfriendly duplicate error: %+v", e)
	}
	var status string
	db.QueryRow(`SELECT r.status FROM responses r JOIN events e ON e.id = r.event_id WHERE e.slug = 'guestbook'`).Scan(&status)
	if status != "in" {
		t.Errorf("append-only answer changed to %q", status)
	}

	// A normal event still lets a name update its answer.
	do(t, "POST", srv.URL+"/events/normal/rsvp", `{"name":"Alex","status":"in"}`)
	if code, b := do(t, "POST", srv.URL+"/events/normal/rsvp", `{"name":"alex","status":"out"}`); code != 200 {
		t.Fatalf("resubmitting on a normal event: %d %s", code, b)
	}
	db.QueryRow(`SELECT r.status FROM responses r JOIN events e ON e.id = r.event_id WHERE e.slug = 'normal'`).Scan(&status)
	if status != "out" {
		t.Errorf("normal event answer = %q, want out", status)
	}
}

func TestRSVPValidation(t *testing.T) {
	db := testDB(t)
	srv := testServer(t, &EventHandlers{DB: db, Blocklist: staticBlocklist("blockedword")})
	insertEvent(t, db, "party", time.Now().Add(24*time.Hour))
	insertEvent(t, db, "demo", time.Now().Add(24*time.Hour))
	db.Exec(`UPDATE events SET is_demo = true WHERE slug = 'demo'`)

	cases := []struct {
		slug, body, wantMsg string
	}{
		{"party", `{"name":"   ","status":"in"}`, "Please enter your name."},
		{"party", `{"name":"` + strings.Repeat("x", 41) + `","status":"in"}`, "Names can be up to 40 characters."},
		{"party", `{"name":"bl0ckedword head","status":"in"}`, "Let's keep it friendly. Try different wording."},
		{"party", `{"name":"Alex","status":"in","guests":21}`, "You can bring up to 20 guests."},
		{"party", `{"name":"Alex","status":"remind_me","notify_via":"not-an-email"}`, "That email address doesn't look right."},
		{"demo", `{"name":"Alex","status":"remind_me","notify_via":"a@example.com"}`, "Reminders are turned off for demo events."},
	}
	for _, c := range cases {
		code, b := do(t, "POST", srv.URL+"/events/"+c.slug+"/rsvp", c.body)
		if code != http.StatusBadRequest {
			t.Errorf("%s: got %d, want 400", c.body, code)
			continue
		}
		if e := decodeError(t, b); e.Error != c.wantMsg {
			t.Errorf("%s: message %q, want %q", c.body, e.Error, c.wantMsg)
		}
	}

	var n int
	db.QueryRow(`SELECT count(*) FROM responses`).Scan(&n)
	if n != 0 {
		t.Errorf("%d invalid RSVPs were saved", n)
	}

	// Trimmed, and the email is dropped for a non-reminder status.
	if code, b := do(t, "POST", srv.URL+"/events/party/rsvp", `{"name":"  Sam  ","status":"in","notify_via":"sam@example.com"}`); code != 200 {
		t.Fatalf("valid RSVP: %d %s", code, b)
	}
	var name string
	var email *string
	db.QueryRow(`SELECT name, notify_via FROM responses`).Scan(&name, &email)
	if name != "Sam" || email != nil {
		t.Errorf("stored name %q email %v, want \"Sam\" and no email", name, email)
	}
}

func TestCreateEventValidationAndDemo(t *testing.T) {
	db := testDB(t)
	srv := testServer(t, &EventHandlers{DB: db, Blocklist: staticBlocklist("blockedword")})

	for body, want := range map[string]string{
		`{"title":"  "}`: "Give your event a name.",
		`{"title":"` + strings.Repeat("x", 81) + `"}`:                     "Event names can be up to 80 characters.",
		`{"title":"Party","location":"` + strings.Repeat("x", 121) + `"}`: "Locations can be up to 120 characters.",
		`{"title":"Party","location":"Blockedword bar"}`:                  "Let's keep it friendly. Try different wording.",
		`{"title":"Party","event_date":"tomorrow"}`:                       "That date or time doesn't look right.",
	} {
		code, b := do(t, "POST", srv.URL+"/events", body)
		if code != http.StatusBadRequest || decodeError(t, b).Error != want {
			t.Errorf("%s: got %d %s, want 400 %q", body, code, b, want)
		}
	}

	code, b := do(t, "POST", srv.URL+"/events", `{"title":"  Ollae Demo ","location":"Alexander Library","event_date":"2026-09-18T12:30:00","emoji":"📚📚","demo":true}`)
	if code != http.StatusCreated {
		t.Fatalf("create demo: %d %s", code, b)
	}
	var ev Event
	json.Unmarshal(b, &ev)
	if !ev.IsDemo || ev.Title != "Ollae Demo" || ev.Emoji != "" {
		t.Errorf("created %+v, want is_demo, trimmed title, invalid emoji dropped", ev)
	}
}

func TestOrganizerDeletesResponse(t *testing.T) {
	db := testDB(t)
	srv := testServer(t, &EventHandlers{DB: db})
	eventID := insertEvent(t, db, "party", time.Now().Add(24*time.Hour))
	insertEvent(t, db, "other", time.Now().Add(24*time.Hour))
	db.Exec(`UPDATE events SET admin_token = slug || '-token'`)
	insertRemindMe(t, db, eventID, "keep")
	insertRemindMe(t, db, eventID, "remove")
	var removeID string
	db.QueryRow(`SELECT id FROM responses WHERE name = 'remove'`).Scan(&removeID)

	for _, token := range []string{"", "wrong", "other-token"} {
		code, _ := do(t, "DELETE", srv.URL+"/events/party/responses/"+removeID+"?admin="+token, "")
		if code != http.StatusUnauthorized && code != http.StatusForbidden {
			t.Errorf("token %q: got %d, want 401/403", token, code)
		}
	}
	if code, _ := do(t, "DELETE", srv.URL+"/events/party/responses/not-a-uuid?admin=party-token", ""); code != 200 {
		t.Errorf("malformed id: got %d, want 200 with nothing deleted", code)
	}

	code, b := do(t, "DELETE", srv.URL+"/events/party/responses/"+removeID+"?admin=party-token", "")
	if code != 200 {
		t.Fatalf("delete: %d %s", code, b)
	}
	var list []PublicResponse
	json.Unmarshal(b, &list)
	if len(list) != 1 || list[0].Name != "keep" {
		t.Errorf("list after delete = %+v, want only keep", list)
	}
}

func TestPerIPRateLimit(t *testing.T) {
	limited := PerIP(3, time.Minute, "slow down")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	hit := func(flyIP, xff string) int {
		r := httptest.NewRequest("POST", "/events", nil)
		r.RemoteAddr = "10.0.0.1:1234" // Fly's proxy
		if flyIP != "" {
			r.Header.Set("Fly-Client-IP", flyIP)
		}
		if xff != "" {
			r.Header.Set("X-Forwarded-For", xff)
		}
		w := httptest.NewRecorder()
		limited.ServeHTTP(w, r)
		if w.Code == http.StatusTooManyRequests {
			if e := decodeError(t, w.Body.Bytes()); e.Error != "slow down" || e.Code != "rate_limited" {
				t.Errorf("429 body = %s", w.Body.Bytes())
			}
		}
		return w.Code
	}

	for i := range 3 {
		// A new X-Forwarded-For each time must not buy a new bucket.
		if code := hit("203.0.113.7", "198.51.100."+string(rune('1'+i))); code != 200 {
			t.Fatalf("request %d: got %d", i+1, code)
		}
	}
	if code := hit("203.0.113.7", "198.51.100.99"); code != http.StatusTooManyRequests {
		t.Errorf("4th request from the same client: got %d, want 429", code)
	}
	if code := hit("203.0.113.8", ""); code != 200 {
		t.Errorf("a different client: got %d, want 200", code)
	}

	// Addresses in one IPv6 /64 share a bucket.
	for i, ip := range []string{"2001:db8:1:2::1", "2001:db8:1:2::2", "2001:db8:1:2:ffff::3"} {
		if code := hit(ip, ""); code != 200 {
			t.Fatalf("IPv6 request %d: got %d", i+1, code)
		}
	}
	if code := hit("2001:db8:1:2:abcd::9", ""); code != http.StatusTooManyRequests {
		t.Errorf("4th request from the same /64: got %d, want 429", code)
	}
}

func TestClaudeDailyCap(t *testing.T) {
	db := testDB(t)
	ctx := t.Context()
	for i := range 3 {
		if err := takeClaudeCall(ctx, db, 3); err != nil {
			t.Fatalf("call %d: %v", i+1, err)
		}
	}
	if err := takeClaudeCall(ctx, db, 3); err != errClaudeCapReached {
		t.Errorf("4th call: %v, want cap reached", err)
	}

	// The count lives in Postgres, so a restarted or second machine sees it.
	srv := testServer(t, &EventHandlers{DB: db, AnthropicKey: "k", ClaudeDailyCap: 3})
	code, b := do(t, "POST", srv.URL+"/parse-event", `{"input":"Volleyball Saturday at 2pm"}`)
	if code != http.StatusServiceUnavailable {
		t.Fatalf("parse over the cap: %d %s", code, b)
	}
	if e := decodeError(t, b); e.Code != "daily_cap" || !strings.Contains(e.Error, "Fill in the details yourself") {
		t.Errorf("cap message = %+v", e)
	}

	// Yesterday's count doesn't carry over.
	db.Exec(`UPDATE claude_usage SET day = day - 1`)
	if err := takeClaudeCall(ctx, db, 3); err != nil {
		t.Errorf("first call of a new day: %v", err)
	}
}

func TestRemindersOff(t *testing.T) {
	db := testDB(t)
	h := &EventHandlers{DB: db, AdminSecret: "secret"}
	srv := testServer(t, h)
	guestbookID := insertEvent(t, db, "guestbook", time.Now().Add(24*time.Hour))
	insertEvent(t, db, "normal", time.Now().Add(24*time.Hour))

	// An email stored before reminders were turned off.
	insertRemindMe(t, db, guestbookID, "early")

	r := httptest.NewRequest("PUT", "/admin/events/guestbook/reminders-off", strings.NewReader(`{"reminders_off":true}`))
	r.Header.Set("Authorization", "Bearer secret")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("slug", "guestbook")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()
	h.AdminSetRemindersOff(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"emails_cleared":1`) {
		t.Fatalf("turn reminders off: %d %s", w.Code, w.Body.String())
	}

	// Remind me is still an answer, without an email.
	if code, b := do(t, "POST", srv.URL+"/events/guestbook/rsvp", `{"name":"Unsure","status":"remind_me"}`); code != 200 {
		t.Fatalf("remind_me without an email: %d %s", code, b)
	}
	// Any email is rejected, whatever the answer.
	for _, body := range []string{
		`{"name":"Sam","status":"remind_me","notify_via":"sam@example.com"}`,
		`{"name":"Sam","status":"in","notify_via":"sam@example.com"}`,
	} {
		code, b := do(t, "POST", srv.URL+"/events/guestbook/rsvp", body)
		if code != http.StatusBadRequest || !strings.Contains(decodeError(t, b).Error, "doesn't send reminders") {
			t.Errorf("%s: %d %s, want 400", body, code, b)
		}
	}

	var stored int
	db.QueryRow(`SELECT count(*) FROM responses WHERE event_id = $1 AND notify_via IS NOT NULL`, guestbookID).Scan(&stored)
	if stored != 0 {
		t.Errorf("%d emails stored on a reminders-off event", stored)
	}
	var status string
	db.QueryRow(`SELECT status FROM responses WHERE name = 'Unsure'`).Scan(&status)
	if status != "remind_me" {
		t.Errorf("status = %q, want remind_me", status)
	}

	// Other events still need an email for Remind me.
	if code, _ := do(t, "POST", srv.URL+"/events/normal/rsvp", `{"name":"Unsure","status":"remind_me"}`); code != http.StatusBadRequest {
		t.Errorf("normal event remind_me without email: %d, want 400", code)
	}

	// The public event says reminders are off; the reminder job skips it even
	// if an email somehow got stored.
	_, body := do(t, "GET", srv.URL+"/events/guestbook", "")
	if !strings.Contains(string(body), `"reminders_off":true`) || strings.Contains(string(body), "notify_via") {
		t.Errorf("public event JSON: %s", body)
	}
	db.Exec(`UPDATE responses SET notify_via = 'sneaky@example.com' WHERE name = 'Unsure'`)
	sent := fakeResend(t, http.StatusOK)
	SendReminders(db, "test-key")
	if got := sent(); len(got) != 0 {
		t.Errorf("reminder job emailed %v on a reminders-off event", got)
	}
}
