package internal

import (
	"net/http"
	"net/url"
	"strings"
)

// GuestbookSlug is the permanent event embedded in the portfolio case study.
// frontend/vercel.json names the same slug; a test keeps them in step.
const GuestbookSlug = "wssrfd7v"

// EmbedParents are the only pages allowed to frame Ollae.
var EmbedParents = []string{
	"https://www.matthewclau.com",
	"https://matthewclau.com",
	"http://localhost:4321",
}

var (
	cspNoFraming   = "frame-ancestors 'none'"
	cspEmbedParent = "frame-ancestors " + strings.Join(EmbedParents, " ")
)

// NoFraming sends frame-ancestors 'none' on every response. A handler that
// may be framed replaces it with AllowEmbedParents rather than adding a
// second header, since browsers enforce every CSP header they receive.
func NoFraming(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", cspNoFraming)
		next.ServeHTTP(w, r)
	})
}

// embeddable reports whether this request for an event page may be framed:
// the guestbook, in embed mode, and never the organizer's edit link.
func embeddable(slug string, q url.Values) bool {
	return slug == GuestbookSlug && q.Get("embed") == "1" && !q.Has("admin")
}

func AllowEmbedParents(w http.ResponseWriter) {
	w.Header().Set("Content-Security-Policy", cspEmbedParent)
}
