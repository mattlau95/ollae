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
| Hosting | Vercel (frontend) · Fly.io (backend + Postgres) |
| Design | Figma · Inter · realfavicongenerator.net |

---

## Architecture

```
ollae/
├── frontend/        # React + Vite SPA → deployed to Vercel
│   ├── src/
│   │   ├── pages/
│   │   │   ├── CreatePage.tsx   # Event creation + share link
│   │   │   └── RSVPPage.tsx     # Event view + RSVP form + activity feed
│   │   └── App.tsx              # React Router setup
│   └── public/                  # Favicon, logo, web manifest
└── backend/         # Go REST API → deployed to Fly.io
    ├── cmd/main.go              # Server setup, routing, CORS
    └── internal/
        ├── events.go            # Handlers: create, get, update event, RSVP upsert
        └── db.go                # Postgres connection
```

**Request flow:**
```
Browser → ollae.app (Vercel) → ollae-backend.fly.dev (Fly.io) → Postgres (Fly.io)
```

**Routing:**
- `GET  /events/:slug`       — fetch event + all responses
- `POST /events`             — create event, returns generated slug
- `PATCH /events/:slug`      — update event title/date/location
- `POST /events/:slug/rsvp`  — upsert RSVP (case-insensitive name dedup)

---

## Design Decisions

**Name-only identity.** No accounts, no sessions. Participants identify by name — "Matt", "Big Mike", "Coach" all valid. The backend upserts on `lower(name)` so resubmitting updates rather than duplicates.

**"Remind me" as an escape hatch, not a peer option.** Three equal RSVP choices (in/maybe/out) invite hedging. "Remind me" is a quiet text link below the two primary buttons — a genuine indecision state, not a fence-sitter button.

**Link permanence.** Editing an event updates it in place. The slug never changes. Any RSVPs already collected stay connected.

**Stateless change flow.** No session memory means no "edit your RSVP" button. The success screen tells participants: reopen the link, resubmit your name. The upsert handles the rest.

---

## Local Development

**Backend:**
```bash
cd backend
go run ./cmd/main.go
# Runs on :8080
# Requires DATABASE_URL env var or local Postgres
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
