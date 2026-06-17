package internal

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"image/png"
	"math"
	"net/http"
	"strings"

	"github.com/fogleman/gg"
	"github.com/go-chi/chi/v5"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

//go:embed assets/Inter-Bold.ttf
var interBoldTTF []byte

//go:embed assets/Inter-Medium.ttf
var interMediumTTF []byte

//go:embed assets/ollae-logo.png
var ollaeLogo []byte

//go:embed assets/OpenMoji-black-glyf.ttf
var openMojiTTF []byte

const ogW, ogH, ogPad = 1200.0, 630.0, 48.0

// DPI 72 = 1pt matches 1px, aligning with Figma's font-size values.
func makeface(ttf []byte, size float64) font.Face {
	f, _ := opentype.Parse(ttf)
	face, _ := opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72})
	return face
}

// capCenter returns the y-baseline needed to visually center capital-letter text
// within a box of height boxH starting at boxY.
func capCenter(face font.Face, boxY, boxH float64) float64 {
	m := face.Metrics()
	capH := float64(m.CapHeight) / 64.0
	if capH <= 0 {
		// Fallback: estimate cap height as 73% of ascent
		capH = float64(m.Ascent) / 64.0 * 0.73
	}
	return boxY + (boxH+capH)/2
}

// tintAmber replaces non-transparent pixels with the ollae amber colour (#F59E0B).
func tintAmber(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := src.At(x, y).RGBA()
			dst.Set(x, y, color.RGBA{R: 245, G: 158, B: 11, A: uint8(a >> 8)})
		}
	}
	return dst
}

// scaleImage scales src to the given width/height using bilinear interpolation.
func scaleImage(src image.Image, w, h int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

func (h *EventHandlers) OGImage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	var event Event
	err := h.DB.QueryRow(`
		SELECT id, slug, title, location, event_date, created_at, emoji
		FROM events WHERE slug = $1
	`, slug).Scan(&event.ID, &event.Slug, &event.Title, &event.Location, &event.EventDate, &event.CreatedAt, &event.Emoji)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	dc := gg.NewContext(int(ogW), int(ogH))
	dc.SetHexColor("#0F172A")
	dc.Clear()

	bold := func(size float64) font.Face { return makeface(interBoldTTF, size) }
	med := func(size float64) font.Face { return makeface(interMediumTTF, size) }

	// Title sizing (Figma: 72px bold)
	titleSize := 72.0
	if len([]rune(event.Title)) > 20 {
		titleSize = 52.0
	}

	// ── Measure section heights ──────────────────────────────────────────
	// MeasureString height = font line-height (used to compute layout gaps).

	dc.SetFontFace(bold(48))
	_, logoLineH := dc.MeasureString("ollae")

	dc.SetFontFace(bold(titleSize))
	_, titleLineH := dc.MeasureString("M")

	dc.SetFontFace(med(24))
	_, tagLineH := dc.MeasureString("M")
	_, labelLineH := dc.MeasureString("M")

	dc.SetFontFace(med(36))
	_, valueLineH := dc.MeasureString("M")

	// Figma section heights (column, justify-content: space-between):
	//   sec1 — logo row (~52px from Figma SVG height)
	//   sec2 — title + tagline (gap: 20px)
	//   sec3 — spacer (72px fixed)
	//   sec4 — meta + button row
	sec1H := math.Max(logoLineH, 52.0)
	sec2H := titleLineH + 20 + tagLineH
	sec3H := 72.0

	hasDate := event.EventDate != nil
	hasLoc := event.Location != ""

	metaItemH := labelLineH + 12 + valueLineH
	var metaH float64
	if hasDate && hasLoc {
		metaH = metaItemH*2 + 32
	} else if hasDate || hasLoc {
		metaH = metaItemH
	}
	const btnH = 104.0
	sec4H := math.Max(metaH, btnH)

	// Space-between: distribute remaining height equally across 3 gaps.
	innerH := ogH - ogPad*2 // 534px
	totalH := sec1H + sec2H + sec3H + sec4H
	gap := (innerH - totalH) / 3.0

	sec1Y := ogPad
	sec2Y := sec1Y + sec1H + gap
	sec4Y := sec2Y + sec2H + gap + sec3H + gap
	// sec4Y + sec4H always equals ogH - ogPad = 582

	// ── LOGO: ollae-logo.png tinted amber, scaled to Figma 154×52px ──────
	if rawLogo, _, err := image.Decode(bytes.NewReader(ollaeLogo)); err == nil {
		// Scale 564×192 → 154×52 (Figma dimensions)
		scaled := scaleImage(tintAmber(rawLogo), 154, 52)
		// Draw top-aligned within sec1 (center vertically if sec1H > 52)
		logoY := sec1Y + (sec1H-52)/2
		dc.DrawImage(scaled, int(ogPad), int(logoY))
	}

	// ── TITLE (left-aligned) ─────────────────────────────────────────────
	boldTitle := bold(titleSize)
	dc.SetFontFace(boldTitle)
	dc.SetHexColor("#F8FAFC")
	titleBaseline := capCenter(boldTitle, sec2Y, titleLineH)
	dc.DrawStringAnchored(truncate(event.Title, 28), ogPad, titleBaseline, 0, 0)

	// ── TAGLINE (left-aligned) ───────────────────────────────────────────
	medFace24 := med(24)
	dc.SetFontFace(medFace24)
	dc.SetHexColor("#94A3B8")
	tagBaseline := capCenter(medFace24, sec2Y+titleLineH+20, tagLineH)
	dc.DrawStringAnchored("Tap to see who's coming →", ogPad, tagBaseline, 0, 0)

	// ── RSVP BUTTON (bottom-right, pill, 104px tall) ─────────────────────
	boldFace40 := bold(40)
	dc.SetFontFace(boldFace40)
	btnText := "RSVP Now →"
	btnTW, _ := dc.MeasureString(btnText)
	const btnPadX = 56.0
	btnW := btnTW + btnPadX*2
	btnX := ogW - ogPad - btnW
	btnY := sec4Y + sec4H - btnH

	dc.SetHexColor("#F59E0B")
	dc.DrawRoundedRectangle(btnX, btnY, btnW, btnH, btnH/2)
	dc.Fill()
	dc.SetHexColor("#000000")
	// Use cap-height centering so text is optically centered in the pill
	dc.DrawStringAnchored(btnText, btnX+btnW/2, capCenter(boldFace40, btnY, btnH), 0.5, 0)

	// ── META: WHEN / WHERE stacked, bottom-aligned ────────────────────────
	xLeft := ogPad
	mb := sec4Y + sec4H // bottom of section = ogH - ogPad

	drawLabel := func(text string, cy float64) {
		face := med(24)
		dc.SetFontFace(face)
		dc.SetHexColor("#64748B")
		dc.DrawStringAnchored(text, xLeft, capCenter(face, cy-labelLineH, labelLineH), 0, 0)
	}
	drawValue := func(text string, cy float64) {
		face := med(36)
		dc.SetFontFace(face)
		dc.SetHexColor("#94A3B8")
		dc.DrawStringAnchored(text, xLeft, capCenter(face, cy-valueLineH, valueLineH), 0, 0)
	}

	if hasDate && hasLoc {
		// Build from bottom up
		whereValTop := mb - valueLineH
		whereLabelTop := whereValTop - 12 - labelLineH
		whenValTop := whereLabelTop - 32 - valueLineH
		whenLabelTop := whenValTop - 12 - labelLineH

		drawLabel("WHEN", whenLabelTop+labelLineH)
		drawValue(formatOGWhen(event), whenValTop+valueLineH)
		drawLabel("WHERE", whereLabelTop+labelLineH)
		drawValue(truncate(event.Location, 28), whereValTop+valueLineH)
	} else if hasDate {
		whenValTop := mb - valueLineH
		whenLabelTop := whenValTop - 12 - labelLineH
		drawLabel("WHEN", whenLabelTop+labelLineH)
		drawValue(formatOGWhen(event), whenValTop+valueLineH)
	} else if hasLoc {
		whereValTop := mb - valueLineH
		whereLabelTop := whereValTop - 12 - labelLineH
		drawLabel("WHERE", whereLabelTop+labelLineH)
		drawValue(truncate(event.Location, 28), whereValTop+valueLineH)
	}

	// ── EMOJI: faded background + badge pill ────────────────────────────
	if event.Emoji != "" {
		emojiFace540 := makeface(openMojiTTF, 540.0)
		emojiFace48 := makeface(openMojiTTF, 48.0)

		// Background: 540px emoji, 17% opacity, right-side
		dc.SetFontFace(emojiFace540)
		_, bgH := dc.MeasureString(event.Emoji)
		dc.SetRGBA255(248, 250, 252, 43) // #F8FAFC at ~17% opacity
		dc.DrawStringAnchored(event.Emoji, ogW-ogPad/2, (ogH+bgH)/2, 1.0, 0)

		// Badge: 96×96 perfect circle, top-right, centered in sec1 row
		dc.SetFontFace(emojiFace48)
		const badgeSize = 96.0
		badgeX := ogW - ogPad - badgeSize
		badgeY := sec1Y + (sec1H-badgeSize)/2
		badgeCX := badgeX + badgeSize/2
		badgeCY := badgeY + badgeSize/2

		dc.SetHexColor("#273548")
		dc.DrawCircle(badgeCX, badgeCY, badgeSize/2)
		dc.Fill()

		dc.SetRGBA255(255, 255, 255, 20) // rgba(255,255,255,0.08)
		dc.SetLineWidth(1)
		dc.DrawCircle(badgeCX, badgeCY, badgeSize/2)
		dc.Stroke()

		// Optically center emoji: baseline = circle center + (ascent − descent) / 2
		m48 := emojiFace48.Metrics()
		ascent48 := float64(m48.Ascent) / 64.0
		descent48 := float64(m48.Descent) / 64.0
		dc.SetHexColor("#F8FAFC")
		dc.DrawStringAnchored(event.Emoji, badgeCX, badgeCY+(ascent48-descent48)/2, 0.5, 0)
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400, s-maxage=86400, no-transform")
	png.Encode(w, dc.Image())
}

// isCrawlerUA returns true for known crawlers that execute JS and would follow
// the location.replace redirect, losing the OG tags.
func isCrawlerUA(ua string) bool {
	return strings.Contains(ua, "facebookexternalhit") ||
		strings.Contains(ua, "Facebot") ||
		strings.Contains(ua, "FacebookBot") || // Messenger preview renderer (Chromium-based, executes JS)
		strings.Contains(ua, "meta-externalagent") || // Meta's newer scraper family
		strings.Contains(ua, "Twitterbot") ||
		strings.Contains(ua, "LinkedInBot") ||
		strings.Contains(ua, "WhatsApp") ||
		strings.Contains(ua, "Slackbot") ||
		strings.Contains(ua, "TelegramBot") ||
		strings.Contains(ua, "MicroMessenger")
}

func (h *EventHandlers) OGPreview(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	var event Event
	err := h.DB.QueryRow(`
		SELECT id, slug, title, location, event_date, created_at, emoji
		FROM events WHERE slug = $1
	`, slug).Scan(&event.ID, &event.Slug, &event.Title, &event.Location, &event.EventDate, &event.CreatedAt, &event.Emoji)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	title := escapeHTML(event.Title)
	ogImage := fmt.Sprintf("https://ollae.app/og/%s", slug)
	ogURL := fmt.Sprintf("https://ollae.app/events/%s", slug)
	appURL := fmt.Sprintf("https://ollae.app/events/%s?_src=app", slug)
	desc := "Tap to see who&#39;s coming &#8594;"

	redirectScript := fmt.Sprintf(`<script>location.replace("%s")</script>`, appURL)
	if isCrawlerUA(r.Header.Get("User-Agent")) {
		redirectScript = ""
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width,initial-scale=1" />
  <title>%s · ollae.app</title>
  <meta property="og:title" content="%s" />
  <meta property="og:description" content="%s" />
  <meta property="og:image" content="%s" />
  <meta property="og:image:secure_url" content="%s" />
  <meta property="og:image:type" content="image/png" />
  <meta property="og:image:width" content="1200" />
  <meta property="og:image:height" content="630" />
  <meta property="og:url" content="%s" />
  <meta property="og:type" content="website" />
  <meta property="og:site_name" content="ollae" />
  <meta name="twitter:card" content="summary_large_image" />
  <meta name="twitter:title" content="%s" />
  <meta name="twitter:description" content="%s" />
  <meta name="twitter:image" content="%s" />
  %s
</head>
<body style="margin:0;background:#0F172A;display:flex;align-items:center;justify-content:center;min-height:100vh;font-family:-apple-system,sans-serif;">
  <a href="%s" style="color:#F59E0B;font-size:18px;font-weight:700;text-decoration:none;">Tap to see who's coming →</a>
</body>
</html>`, title, title, desc, ogImage, ogImage, ogURL, title, desc, ogImage, redirectScript, appURL)

	w.Header().Set("Content-Type", "text/html;charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-transform")
	w.Write([]byte(html))
}

func (h *EventHandlers) OGImageJSON(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"image": fmt.Sprintf("https://ollae.app/og/%s", slug),
		"url":   fmt.Sprintf("https://ollae.app/events/%s", slug),
	})
}

func formatOGWhen(event Event) string {
	if event.EventDate == nil {
		return ""
	}
	date := event.EventDate.Format("Mon, Jan 2")
	if event.EventDate.Hour() == 0 && event.EventDate.Minute() == 0 {
		return date
	}
	return date + " · " + event.EventDate.Format("3:04 PM")
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "..."
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
