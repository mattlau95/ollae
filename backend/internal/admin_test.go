package internal

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminAuth(t *testing.T) {
	withAuth := func(value string) *http.Request {
		r := httptest.NewRequest("GET", "/admin/events", nil)
		if value != "" {
			r.Header.Set("Authorization", value)
		}
		return r
	}

	if adminAuth("", withAuth("Bearer ")) {
		t.Error("empty secret must never authorize")
	}
	if adminAuth("s3cret", withAuth("")) {
		t.Error("missing header must not authorize")
	}
	if adminAuth("s3cret", withAuth("Bearer wrong")) {
		t.Error("wrong token must not authorize")
	}
	if adminAuth("s3cret", withAuth("s3cret")) {
		t.Error("token without Bearer prefix must not authorize")
	}
	if !adminAuth("s3cret", withAuth("Bearer s3cret")) {
		t.Error("correct token must authorize")
	}
}
