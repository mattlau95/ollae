# ![ollae](https://raw.githubusercontent.com/mattlau95/ollae/master/frontend/public/ollae-logo.svg)

**ollae** (올래) — Korean for *"wanna come? / you in?"*

A frictionless group RSVP tool. Drop a link in the group chat, see who's in. No accounts, no apps, no friction.

**Live:** [ollae.app](https://ollae.app)

---

## The Problem

Every group activity starts the same way — a message in the chat, a flood of replies, and no clear picture of who's actually coming. Existing tools either require everyone to sign up, feel corporate, or are buried inside another app.

Ollae is just a link. The organizer creates an event and shares the URL. Participants tap it, RSVP, and the list updates in real time.

---

## Features

- **Create an event** — name, date, time, location. Takes 15 seconds.
- **Shareable link** — one URL, works for everyone, no login required.
- **RSVP flow** — I'm in / Can't make it / Remind me closer to the date.
- **Live activity feed** — see who's responded and when, in real time.
- **Case-insensitive upsert** — "Matt" and "matt" are the same person. Re-submitting updates your existing RSVP instead of creating a duplicate.
- **Edit events** — organizer can update details after sharing; the link never changes.
- **Mobile-first** — designed for the group chat context: thumb-zone layout, dark UI, fast load.

---

## Stack

| Layer | Choice |
|---|---|
| Frontend | React 19 · TypeScript · Vite · Tailwind CSS v4 |
| Backend | Go 1.26.3 · chi v5 |
| Database | PostgreSQL 18 · lib/pq |
| AI | Claude Haiku 4.5 (Anthropic Messages API) — natural-language event parsing, called server-side |
| Hosting | Vercel (frontend) · Fly.io (backend + Postgres) |
| Design | Figma · Inter · realfavicongenerator.net |

---

## Architecture

```
ollae/
├── frontend/        # React + Vite SPA → deployed to Vercel
│   ├── src/
│   │   ├── pages/
│   │   │   ├── CreatePage.tsx   # Describe → parse → edit → share link
│   │   │   ├── RSVPPage.tsx     # Event view + RSVP form + activity feed
│   │   │   └── AdminPage.tsx    # Password-gated dashboard: events, deletes, retention
│   │   ├── App.tsx              # React Router setup
│   │   ├── api.ts               # API base URL
│   │   └── celebrate.ts         # Confetti, gated on prefers-reduced-motion
│   ├── vercel.json              # SPA rewrites + crawler routing for /events/:slug
│   └── public/                  # Favicon, logo, web manifest
└── backend/         # Go REST API → deployed to Fly.io
    ├── cmd/main.go              # Server setup, routing, CORS
    ├── docs/ollae-devlog.md     # Session-by-session build log
    └── internal/
        ├── events.go            # Create / get / update event, RSVP upsert
        ├── parse.go             # Natural-language event parsing via the Claude API
        ├── og.go                # OG image rendering + crawler-facing preview page
        ├── reminders.go         # "Remind me" emails (Resend), cron-triggered
        ├── admin.go             # Bearer-token admin endpoints, retention setting
        └── db.go                # Postgres connection + migrations
```

**Request flow:**
```
Browser → ollae.app (Vercel) → ollae-backend.fly.dev (Fly.io) → Postgres (Fly.io)
```

**Routing:**
- `POST  /parse-event`         — natural language → structured event (Claude)
- `POST  /events`              — create event, returns slug + admin token
- `GET   /events/:slug`        — fetch event + all responses
- `PATCH /events/:slug`        — update event (requires `?admin=<token>`)
- `POST  /events/:slug/rsvp`   — upsert RSVP (case-insensitive name dedup)
- `GET   /og/:slug`            — 1200×630 preview image, rendered server-side
- `GET   /og-preview/:slug`    — crawler-facing HTML with OG tags (see `vercel.json`)
- `GET   /cron/remind`         — send due reminder emails (token-protected)
- `/admin/*`                   — dashboard endpoints, bearer-token auth

**Share-link routing.** A tap on `ollae.app/events/:slug` from a chat app is first served by `/og-preview/:slug`, so link-preview crawlers see static OG tags without running JavaScript. Real browsers are bounced into the SPA with `?_src=app`, which `vercel.json` rewrites to `index.html`.

---

## Design Decisions

**Name-only identity.** No accounts, no sessions. Participants identify by name — "Matt", "Big Mike", "Coach" all valid. The backend upserts on `lower(name)` so resubmitting updates rather than duplicates.

**"Remind me" as an escape hatch, not a peer option.** Three equal RSVP choices (in/maybe/out) invite hedging. "Remind me" is a quiet text link below the two primary buttons — a genuine indecision state, not a fence-sitter button.

**Link permanence.** Editing an event updates it in place. The slug never changes. Any RSVPs already collected stay connected.

**Stateless change flow.** No session memory means no "edit your RSVP" button. The success screen tells participants: reopen the link, resubmit your name. The upsert handles the rest.

**Organizer edit links are bearer tokens in the URL.** The creator gets a second link with `?admin=<token>` that unlocks editing. This is a deliberate trade-off for a no-accounts product: the edit link is exactly as easy to save and share as the event link. The cost is that the token sits in browser history and request logs and can't be rotated — acceptable for low-stakes social plans, not the right model for anything sensitive.

---

## Local Development

**Backend:**
```bash
cd backend
go test ./...          # unit tests, no database needed
go run ./cmd/main.go
# Runs on :8080
# Requires DATABASE_URL, e.g. postgres://user:pass@localhost:5432/ollae?sslmode=disable
# Optional: ANTHROPIC_API_KEY (natural-language parsing; without it the form
#   falls back to manual entry), ADMIN_SECRET (/admin), RESEND_API_KEY + CRON_TOKEN
#   (reminder emails), FB_APP_TOKEN (preview rescrape), FRONTEND_URL (extra CORS origin)
```

**Frontend:**
```bash
cd frontend
npm install
npm run dev
# Runs on :5173
# API auto-routes to localhost:8080 in dev
```

---

## Deployment

- **Frontend** — Vercel, root directory `frontend/`, auto-deploys on push to `master`
- **Backend** — Fly.io, `fly deploy` from `backend/`
- **Database** — Fly Postgres (`showup-db`), attached to `ollae-backend`

---

*Built by [Matt Lau](https://github.com/mattlau95)*
