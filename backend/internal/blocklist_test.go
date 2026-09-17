package internal

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// staticBlocklist builds a Blocklist without a database, for tests.
func staticBlocklist(terms ...string) *Blocklist {
	b := &Blocklist{terms: [][]string{}}
	for _, t := range terms {
		b.terms = append(b.terms, normalizeWords(t))
	}
	return b
}

func TestBlocklistMatching(t *testing.T) {
	b := staticBlocklist("blockedword", "sillyterm", "two words")

	match := []string{
		"blockedword",
		"BlockedWord", "BLOCKEDWORD", // casing
		"bl0ck3dw0rd", "s1llyterm", "$!llyterm", "5illyterm", // character swaps
		"blockedword!", "blockedword!!!", "(blockedword)", "blockedword's", "Hi, blockedword.", // punctuation
		"b.l.o.c.k.e.d.w.o.r.d", "b l o c k e d w o r d", "b-l-o-c-k-e-d-w-o-r-d", // spelled out
		"Two Words", "two-words", "all two   words here", // phrase
	}
	noMatch := []string{
		"blockedwordsmith", "unblockedword", "blockedwords", "sillyterms", // longer words containing a term
		"twowords", "two", "words",
		"Alexander Library", "Board game night", "", "a b c",
	}
	for _, s := range match {
		if !b.Matches(s) {
			t.Errorf("Matches(%q) = false, want true", s)
		}
	}
	for _, s := range noMatch {
		if b.Matches(s) {
			t.Errorf("Matches(%q) = true, want false", s)
		}
	}
}

func TestEmptyBlocklistAllowsEverything(t *testing.T) {
	var nilList *Blocklist
	for _, b := range []*Blocklist{nilList, staticBlocklist()} {
		for _, s := range []string{"blockedword", "anything at all", ""} {
			if b.Matches(s) {
				t.Errorf("empty blocklist matched %q", s)
			}
		}
	}
}

func TestNormalizeTerm(t *testing.T) {
	for in, want := range map[string]string{
		"BL0CKEDWORD":        "blockedword",
		"  Two   Words!  ":   "two words",
		"b.l.o.c.k.e.d.word": "blocked word",
		"!!!":                "",
		"   ":                "",
	} {
		if got := NormalizeTerm(in); got != want {
			t.Errorf("NormalizeTerm(%q) = %q, want %q", in, got, want)
		}
	}
}

func adminDo(t *testing.T, h *EventHandlers, handler http.HandlerFunc, auth, body string) (int, []byte) {
	t.Helper()
	r := httptest.NewRequest("POST", "/admin/blocked-terms", strings.NewReader(body))
	if auth != "" {
		r.Header.Set("Authorization", "Bearer "+auth)
	}
	w := httptest.NewRecorder()
	handler(w, r)
	return w.Code, w.Body.Bytes()
}

func TestBlockedTermsAdminAndRefresh(t *testing.T) {
	db := testDB(t)
	RunMigrations(db)
	h := &EventHandlers{DB: db, AdminSecret: "secret", Blocklist: NewBlocklist(db)}
	otherMachine := NewBlocklist(db)
	srv := testServer(t, h)
	insertEvent(t, db, "party", time.Now().Add(24*time.Hour))

	if code, _ := adminDo(t, h, h.AdminAddBlockedTerm, "wrong", `{"term":"blockedword"}`); code != http.StatusUnauthorized {
		t.Errorf("add without the admin secret: %d, want 401", code)
	}
	if code, _ := adminDo(t, h, h.AdminAddBlockedTerm, "secret", `{"term":"  !!! "}`); code != http.StatusBadRequest {
		t.Errorf("add an empty term: %d, want 400", code)
	}

	// Empty list: everything gets through.
	if code, b := do(t, "POST", srv.URL+"/events/party/rsvp", `{"name":"Blockedword Fan","status":"in"}`); code != 200 {
		t.Fatalf("RSVP with an empty list: %d %s", code, b)
	}

	code, body := adminDo(t, h, h.AdminAddBlockedTerm, "secret", `{"term":"  BL0CKEDWORD "}`)
	var list struct{ Terms []string }
	json.Unmarshal(body, &list)
	if code != 200 || len(list.Terms) != 1 || list.Terms[0] != "blockedword" {
		t.Fatalf("add: %d %s, want the normalized term listed", code, body)
	}

	// This machine applies it immediately.
	code, body = do(t, "POST", srv.URL+"/events/party/rsvp", `{"name":"Blockedword Fan","status":"in"}`)
	if code != http.StatusBadRequest || decodeError(t, body).Error != "Let's keep it friendly. Try different wording." {
		t.Errorf("RSVP with a blocked term: %d %s", code, body)
	}
	// The other machine picks it up on its next refresh.
	if otherMachine.Matches("blockedword") {
		t.Error("other machine matched before refreshing")
	}
	if err := otherMachine.Refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	if !otherMachine.Matches("blockedword") {
		t.Error("other machine didn't pick up the new term")
	}

	if code, body := adminDo(t, h, h.AdminRemoveBlockedTerm, "secret", `{"term":"blockedword"}`); code != 200 || !strings.Contains(string(body), `"terms":[]`) {
		t.Errorf("remove: %d %s", code, body)
	}
	otherMachine.Refresh(t.Context())
	if h.Blocklist.Matches("blockedword") || otherMachine.Matches("blockedword") {
		t.Error("term still blocked after removal")
	}
}

func TestRejectedTextIsNeverLogged(t *testing.T) {
	db := testDB(t)
	var logs bytes.Buffer
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	h := &EventHandlers{DB: db, Blocklist: staticBlocklist("blockedword")}
	srv := testServer(t, h)
	insertEvent(t, db, "party", time.Now().Add(24*time.Hour))

	secret := "Distinctive Blockedword Visitor Text"
	do(t, "POST", srv.URL+"/events/party/rsvp", `{"name":"`+secret+`","status":"in"}`)
	do(t, "POST", srv.URL+"/events", `{"title":"`+secret+`"}`)
	do(t, "PATCH", srv.URL+"/events/party?admin=x", `{"title":"`+secret+`"}`)

	if strings.Contains(strings.ToLower(logs.String()), "distinctive") {
		t.Errorf("rejected text reached the log:\n%s", logs.String())
	}
}
