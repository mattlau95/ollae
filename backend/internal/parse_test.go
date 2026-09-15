package internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestParseRetryAfter(t *testing.T) {
	cases := map[string]time.Duration{
		"":    0,
		"abc": 0,
		"-1":  0,
		"2":   2 * time.Second,
		"600": maxRetryAfter,
	}
	for in, want := range cases {
		if got := parseRetryAfter(in); got != want {
			t.Errorf("parseRetryAfter(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestCallAnthropicRetriesThenSucceeds(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if r.Header.Get("x-api-key") != "k" {
			t.Errorf("missing api key header")
		}
		switch n {
		case 1:
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
		case 2:
			w.WriteHeader(http.StatusBadGateway)
		default:
			w.Write([]byte(`{"content":[{"text":"{}"}]}`))
		}
	}))
	defer srv.Close()

	orig := anthropicURL
	anthropicURL = srv.URL
	defer func() { anthropicURL = orig }()

	status, body, err := callAnthropic(context.Background(), "k", []byte(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", status, body)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Errorf("calls = %d, want 3", got)
	}
}

func TestCallAnthropicGivesUpOn429(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	orig := anthropicURL
	anthropicURL = srv.URL
	defer func() { anthropicURL = orig }()

	status, _, err := callAnthropic(context.Background(), "k", []byte(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != http.StatusTooManyRequests {
		t.Errorf("status = %d, want 429", status)
	}
	if got := atomic.LoadInt32(&calls); got != anthropicAttempts {
		t.Errorf("calls = %d, want %d", got, anthropicAttempts)
	}
}

func TestCallAnthropicDoesNotRetryClientErrors(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	orig := anthropicURL
	anthropicURL = srv.URL
	defer func() { anthropicURL = orig }()

	status, _, _ := callAnthropic(context.Background(), "k", []byte(`{}`))
	if status != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", status)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("calls = %d, want 1 (no retry on 4xx)", got)
	}
}
