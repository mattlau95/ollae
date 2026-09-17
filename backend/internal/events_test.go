package internal

import (
	"encoding/json"
	"strings"
	"testing"
)

// Anyone with an event link can read GET /events/:slug and the RSVP response,
// so the type they return must never carry reminder emails.
func TestPublicResponseOmitsNotifyVia(t *testing.T) {
	b, err := json.Marshal(PublicResponse{ID: "1", Name: "Alex", Status: "remind_me"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "notify_via") {
		t.Fatalf("public response JSON contains notify_via: %s", b)
	}
}
