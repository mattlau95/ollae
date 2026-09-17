package internal

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAppURLKeepsEveryQueryParameter(t *testing.T) {
	q := url.Values{"embed": {"1"}, "utm_source": {"portfolio"}, "_src": {"old"}}
	got, err := url.Parse(appURL("wssrfd7v", q))
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != "/events/wssrfd7v" {
		t.Errorf("path = %q", got.Path)
	}
	want := url.Values{"embed": {"1"}, "utm_source": {"portfolio"}, "_src": {"app"}}
	if got.Query().Encode() != want.Encode() {
		t.Errorf("query = %q, want %q", got.RawQuery, want.Encode())
	}
}

// cspLines returns every Content-Security-Policy header on a response.
func cspLines(h http.Header) []string { return h.Values("Content-Security-Policy") }

func TestOGPreviewFramingAndRedirect(t *testing.T) {
	db := testDB(t)
	srv := testServer(t, &EventHandlers{DB: db})
	insertEvent(t, db, GuestbookSlug, time.Now().Add(24*time.Hour))
	insertEvent(t, db, "otherevt", time.Now().Add(24*time.Hour))

	allow := "frame-ancestors https://www.matthewclau.com https://matthewclau.com http://localhost:4321"
	none := "frame-ancestors 'none'"
	cases := []struct {
		path, want string
	}{
		{"/og-preview/" + GuestbookSlug + "?embed=1", allow},
		{"/og-preview/" + GuestbookSlug, none},
		{"/og-preview/" + GuestbookSlug + "?embed=0", none},
		{"/og-preview/" + GuestbookSlug + "?embed=1&admin=tok", none},
		{"/og-preview/otherevt?embed=1", none},
		{"/og-preview/missing1?embed=1", none},
		{"/events/otherevt", none},
	}
	for _, c := range cases {
		resp, err := noRedirects.Get(srv.URL + c.path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if lines := cspLines(resp.Header); len(lines) != 1 || lines[0] != c.want {
			t.Errorf("%s: CSP headers %q, want exactly [%q]", c.path, lines, c.want)
		}
	}

	// The redirect in the page keeps embed=1 and can't be broken out of.
	resp, err := noRedirects.Get(srv.URL + "/og-preview/" + GuestbookSlug + `?embed=1&x=%22%3C%2Fscript%3E`)
	if err != nil {
		t.Fatal(err)
	}
	body := make([]byte, 8192)
	n, _ := resp.Body.Read(body)
	resp.Body.Close()
	page := string(body[:n])
	if !strings.Contains(page, `embed=1`) || !strings.Contains(page, `_src=app`) {
		t.Errorf("redirect lost query parameters:\n%s", page)
	}
	if strings.Contains(page, `</script>"`) || strings.Count(page, "</script>") != 1 {
		t.Errorf("query parameter escaped the script:\n%s", page)
	}
	if resp.Header.Get("X-Robots-Tag") != "noindex" || !strings.Contains(page, `name="robots" content="noindex"`) {
		t.Error("guestbook page is not noindex")
	}
}

// vercel.json frames the same slug on the app side; the two must agree.
func TestVercelConfigNamesGuestbookSlug(t *testing.T) {
	raw, err := os.ReadFile("../../frontend/vercel.json")
	if err != nil {
		t.Skip("frontend/vercel.json not found")
	}
	var cfg struct {
		Headers []struct {
			Source  string `json:"source"`
			Headers []struct{ Key, Value string }
		} `json:"headers"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, rule := range cfg.Headers {
		for _, h := range rule.Headers {
			if h.Key == "Content-Security-Policy" && h.Value == cspEmbedParent {
				found = true
				if rule.Source != "/create" && rule.Source != "/events/"+GuestbookSlug {
					t.Errorf("allowlist on unexpected source %q", rule.Source)
				}
			}
		}
	}
	if !found || !strings.Contains(string(raw), `"/events/`+GuestbookSlug+`"`) {
		t.Errorf("vercel.json doesn't allow framing /events/%s with %q", GuestbookSlug, cspEmbedParent)
	}
}
