# Showup.gg — Dev Log
## Session 01 — May 14, 2026

---

### What We Built

A fully working Go backend for Showup.gg, connected to a real PostgreSQL database, with three live API endpoints. Starting from zero — no Go installed, no database, no project folder.

---

### The Stack

| Layer | Choice |
|---|---|
| Language | Go 1.26.3 |
| Router | chi v5.2.5 |
| Database | PostgreSQL 18 |
| DB Driver | lib/pq |
| Slug generation | go-nanoid v2 |
| API testing | Thunder Client (VS Code) |

---

### Issues Closed

**MAT-49 — Go project scaffold**
Installed Go, initialized the module, installed chi, set up `cmd/internal/db` folder structure, added request logging and panic recovery middleware, and built a `/health` endpoint. First time running `go run` and seeing the logger print live request output.

**MAT-50 — PostgreSQL schema**
Installed PostgreSQL 18, created the `showup` database, and defined the `events` and `responses` tables. Hit a real PostgreSQL gotcha: expression-based unique indexes (`lower(name)`) can't be created as named constraints — had to use `CREATE UNIQUE INDEX` instead, which came back to bite us in MAT-52.

Also made two meaningful product decisions during schema work:

1. **Dropped "maybe" as an RSVP status.** After researching the UX implications — commitment psychology, decision fatigue literature — we reframed "maybe" as "Remind Me": a notification intent rather than an attendance hedge. This is a genuinely different product surface. Schema updated: status now constrained to `in`, `out`, `remind_me`. A nullable `notify_via` column added to `responses` for email/phone.

2. **Created MAT-56** (Phase 2) to track the full "Remind Me" notification system — provider selection, scheduler, frontend reveal pattern.

**MAT-51 — POST /events and GET /events/:slug**
First endpoints that connect Go to Postgres. Introduced the `EventHandlers` struct pattern — dependency injection for the DB connection. `POST /events` generates an 8-char nanoid slug and inserts a new event. `GET /events/:slug` fetches the event and all its responses in a single handler. Both tested in Thunder Client with real data.

**MAT-52 — POST /events/:slug/rsvp**
The most important endpoint — the RSVP upsert. Hit the constraint naming issue from MAT-50: `ON CONFLICT ON CONSTRAINT unique_name_per_event` failed because expression indexes aren't named constraints in Postgres. Fixed by switching to `ON CONFLICT (event_id, lower(name))` directly. Verified case-insensitive upsert: submitting as "matt" correctly updated the existing "Matt" row rather than creating a duplicate.

---

### Product Decisions Made

| Decision | Rationale |
|---|---|
| Dropped "maybe" status | Low-signal for organizer, low-commitment for participant. Replaced with "Remind Me" — a notification intent with a different data shape. |
| `notify_via` nullable column | Baked Phase 2 notification infrastructure into the schema now. Cheap to add, painful to retrofit. |
| Name-only identity (no accounts) | Zero friction for participants. Acceptable trust model for a private link-based tool. |
| Single `name` text field | "Matt", "Big Mike", "Coach" all valid. Informal group, informal identity. |
| Common name hint (MAT-57) | Frontend-only UX nudge — "Add a last initial so others know it's you." Non-blocking, friendly copy. |

---

### New Issues Created

| Issue | Title | Priority |
|---|---|---|
| MAT-56 | Phase 2 — "Remind Me" notification system | Medium |
| MAT-57 | Frontend — name input hint for common names | Medium |
| MAT-58 | Design — RSVP page redesign (Figma) | High |
| MAT-59 | Design — OG preview card + event creation page | Medium |
| MAT-60 | Design — design system tokens (colors, type, spacing) | Medium |

---

### Things That Tripped Us Up (and Why They Matter)

**Expression indexes vs. named constraints in PostgreSQL**
Standard `UNIQUE` constraints work on raw column values. When you need case-insensitive uniqueness via `lower(name)`, you need a `CREATE UNIQUE INDEX` instead. The catch: Postgres's `ON CONFLICT ON CONSTRAINT` syntax only works with named constraints, not expression indexes. The fix is `ON CONFLICT (event_id, lower(name))` — referencing the expression directly. A subtle but real distinction worth remembering.

**PATH setup on Windows**
Go and PostgreSQL both required manual PATH configuration in PowerShell after installation. `$env:Path +=` sets it for the current session only — a permanent fix requires system environment variable settings. Worth noting for any Windows-based Go setup.

**Port conflict**
Restarting the server without killing the previous process hits `bind: Only one usage of each socket address`. Always `Ctrl+C` the running server before `go run`.

---

### What's Next

**Immediate (backend)**
- MAT-53 — Scaffold React + Vite frontend, wire to real API
- MAT-54 — Deploy backend to Fly.io
- MAT-55 — Deploy frontend to Vercel

**Design queue (your lane)**
- MAT-58 — RSVP page redesign in Figma (start here)
- MAT-60 — Design system tokens (build alongside MAT-58)
- MAT-59 — OG preview card + event creation page

**Phase 2**
- MAT-56 — "Remind Me" notification system
- MAT-57 — Common name hint in frontend

---

### Reflection

The backend is real. Three endpoints, a live database, upsert logic, case-insensitive uniqueness — all tested against actual data. The schema decisions made today (especially the "Remind Me" reframe) came from genuine product thinking, not just copying the roadmap. That's the kind of decision-making worth talking about in a case study or interview: identifying that "maybe" is a different UX problem than attendance ambiguity, and solving it at the data model level before writing a single line of frontend code.

The next session is a context switch — Figma for design, then wiring the frontend to the API you just built.

---

*Stack: Go 1.26.3 · chi v5 · PostgreSQL 18 · lib/pq · go-nanoid v2*
*Tools: VS Code · Thunder Client · psql · Linear*

---

## Session 02 — May 15, 2026

---

### What We Built

A complete 3-frame RSVP flow in Figma, a token-based design system (Variables), and a Figma Professional upgrade to support real design system work.

---

### Design System Setup (MAT-60)

Upgraded to Figma Professional to unlock Variables — the current industry standard for design token work. Set up three collections:

**Colors** — 20 variables covering:
- 4 background surface levels (`bg/base` through `bg/overlay`)
- 2 accent colors (amber primary + secondary)
- 3 status states with bg/border/text each (in, out, remind)
- 4 text levels (primary through disabled)
- 2 border tokens (default + focus)

**Spacing** — 9 number variables on an 8pt scale (`space/1` through `space/12`)

**Radius** — 5 number variables (`radius/sm` through `radius/full`)

The token naming convention uses `/` for grouping — `status/in`, `status/in/bg`, `status/in/border` — which creates nested folders in Figma's panel and maps cleanly to Tailwind config during implementation.

---

### RSVP Page Design (MAT-58)

Designed three frames at iPhone 14 Pro dimensions (393×852px) with auto layout, representing the complete participant flow:

**Frame 1 — Default state**
Empty name field, neutral unselected buttons, dark/disabled submit. The first-time view. Event info at top, activity feed in middle, RSVP form pinned to lower half via auto layout spacer for thumb-zone accessibility.

**Frame 2 — Selected state**
Name filled ("Matthew L"), "I'm in" button in green selected variant, amber Submit button active. Represents the "about to submit" moment — the most important state in the flow.

**Frame 3 — Success state**
Standalone confirmation screen, deliberately separate from the main flow. Large emoji, "You're on the list!", personalized status line in green, reopen-link copy for returning users, "← View who's coming" link back to the feed. Vertically centered, clean, receipt-like.

---

### Key Design Decisions

**Activity feed instead of avatar list**
Replaced the static roster with a timestamped text feed ("Matt T is in 🏐 · 2m ago"). Reads like a group chat — native to the context, reinforces the "real people, live activity" feeling. Better social proof than a static list.

**"17 attending" as hero number**
Single large number replaces the three-counter row (Going / Maybe / Total) from the prototype. More immediate, stronger social proof signal.

**"Remind Me" as a secondary text link, not a third button**
"Not sure yet — remind me closer to the date 🔔" sits below the two primary buttons as a quiet text link. Forces a real binary decision first (in or out), treats indecision as an escape hatch rather than a peer option. Reduces fence-sitting while still accommodating genuinely undecided people.

**Two primary buttons side by side, not stacked**
I'm in / Can't make it as equal-width horizontal pair. Saves vertical space, presents the choice as a clear binary, works well with thumb reach in the lower half of the screen.

**Submit button state tied to form completion**
Dark/disabled by default. Amber only when name is entered AND a button is tapped. Two component variants in Figma — Default and Active.

**Success screen as its own moment**
No repeated event info on the success screen. Clean, focused, one job — confirmation. Modeled after a checkout receipt: you know where you are, you don't need the full page context again.

**Stateless change flow**
No accounts, no session memory. If someone needs to change their RSVP, they reopen the link and resubmit with the same name — the upsert handles it. Copy on the success screen makes this explicit: "Changed your plans? Just reopen this link and resubmit your name."

---

### UX Research Applied

- **Thumb zone**: All interactive elements (buttons, input, submit) in the lower half of the 852px frame, per Hoober's 2025 touch research (75% single-thumb usage, comfortable reach = bottom third)
- **Dark-first**: Designed dark from the start, not adapted from light — current industry default for mobile
- **4-level surface system**: bg/base → bg/surface → bg/elevated → bg/overlay, not a single dark grey
- **Commitment psychology**: Binary button choice + "remind me" as escape hatch reduces fence-sitting vs. three equal options
- **Token system**: Variables set up to map directly to Tailwind config — same token names in design and code

---

### New Issues Created

| Issue | Title | Priority |
|---|---|---|
| MAT-56 | Phase 2 — "Remind Me" notification system | Medium |
| MAT-57 | Frontend — name input hint for common names | Medium |

---

### Issues Closed

- **MAT-58** — RSVP page redesign (Figma) ✓
- **MAT-60** — Design system tokens ✓

---

### What's Next

**Design**
- MAT-59 — OG preview card (Figma) + event creation page

**Frontend**
- MAT-53 — Scaffold React + Vite, wire to real API using Figma designs

**Backend**
- MAT-54 — Deploy to Fly.io
- MAT-55 — Deploy frontend to Vercel

---

### Reflection

The most interesting design decision this session was the "Remind Me" reframe — moving it from a third equal button to a quiet text link. That came from a real product instinct: two clear choices with an escape hatch is better UX than three equal options that invite hedging. The backend already supports it via the `remind_me` status and `notify_via` column, so design and data model are in sync.

The token system setup also paid off immediately — being able to reference `status/in` instead of `#22c55e` throughout the design file keeps the work coherent and makes the Tailwind implementation straightforward.

---

*Tools: Figma Professional · Variables · Auto Layout · Linear*

---

## Session 03 — May 17, 2026

---

### What We Built

Completed the full design package for Showup.gg — OG preview card (1200×630) and event creation page (2 frames). All design work is now done and ready for frontend implementation.

---

### Issues Closed

**MAT-59 — OG preview card + event creation page**

**OG Card (1200×630)**
Designed from scratch rather than adapting the existing HTML mockup. Key decisions:
- Large faded volleyball emoji filling the right half at ~15% opacity — fills negative space, communicates sport without competing with content
- Small volleyball icon in a circle top right — ties to the large background element
- "WEEKLY PICKUP" amber pill tag top left
- "Pickup Volleyball" at 72px as the hero
- "Tap to see who's coming →" tagline in muted text — curiosity hook that works at every crop size
- WHEN / WHERE meta in bottom left with uppercase labels
- "RSVP Now →" amber CTA button bottom right

Verified readability at iMessage and WhatsApp crop sizes — title and CTA survive even the smallest preview.

**Event Creation Page (2 frames)**

Frame 1 — Default state:
- Header: "Create Event" + "Fill in the details below, get a shareable link."
- Four inputs: Event Name, Date, Time, Location
- Amber "Create Event →" button
- No link card shown — link only appears after submit

Frame 2 — Success state:
- Form at 30% opacity showing the filled details as context
- Button relabeled "Edit Event Details" — correct label for this state
- "Share this link" section revealed below with the generated link and copy action

---

### Key Design Decisions

**OG card right-side treatment**
Chose large faded emoji over diagonal stripe (from original mockup) or empty negative space. The emoji is immediately sport-identifiable at any crop size, ties the small icon in the corner together, and adds depth without color noise.

**Event creation two-state approach**
Rather than a separate success page, Frame 2 keeps the form visible at reduced opacity. This communicates "these are the details that generated this link" — the organizer can see exactly what their event contains without navigating away. Tapping "Edit Event Details" would re-enable the form.

**Form inputs disabled in success state**
Noted for implementation: inputs must be non-interactive in Frame 2 until "Edit Event Details" is tapped. Design shows this via opacity; code needs to enforce it via disabled state.

**Icon library decision**
Selected Lucide for Figma and implementation. Figma plugin installed. Lucide-react already in the planned frontend stack — same icon names in design and code, zero translation cost.

---

### Design System Status

All screens now use tokens consistently:
- Colors: all pulls from Variables collection
- Spacing: 8pt scale throughout
- Radius: component-appropriate tokens
- Typography: consistent scale across all frames

Full design file contains:
- 🎨 Cover
- 📐 Tokens (Variables)
- 📱 RSVP Page (3 frames: Default, Selected, Success)
- 🖼️ OG Card (1 frame at 1200×630)
- ➕ Event Creation (2 frames: Default, Success)

---

### What's Next

All design work complete. Moving to frontend implementation.

- **MAT-53** — Scaffold React + Vite, wire to real API
- **MAT-54** — Deploy backend to Fly.io
- **MAT-55** — Deploy frontend to Vercel

---

### Reflection

The OG card design session surfaced a good lesson about design at scale — dimensions that look right in a lofi mockup at 50% need to be doubled for the actual asset. Everything from padding to font size to icon size needs to account for the real canvas size, not the preview size.

The event creation two-state approach (form at 30% opacity in success state) came from a product instinct: don't hide context after a form submission. The organizer just filled in four fields — keeping them visible confirms what generated the link without requiring navigation. Small decision, good UX.

---

*Tools: Figma Professional · Lucide Icons · Linear*

---

## Session 04 — May 18, 2026

---

### What We Built

A complete React + TypeScript frontend wired to the real Go API — two pages, full RSVP flow, mobile-first layout, activity feed, and success screen.

---

### Issues Closed

**MAT-53 — React + Vite scaffold, wire RSVP page to real API**

Scaffolded with Vite + React + TypeScript + Tailwind. Two pages:

**CreatePage**
Event creation form (title, date, location). On submit, calls `POST /events`, receives the slug, and reveals a copyable shareable link. The form dims to 30% opacity in the success state (matching the Figma design) — the organizer can see what generated the link without navigating away. "Edit Event Details" re-enables the form.

**RSVPPage**
Fetches event on load via `GET /events/:slug`. Renders:
- Event header (title, date with calendar icon, location with pin icon)
- "17 attending" hero count
- Activity feed — last 8 responses in reverse chronological order with relative timestamps (just now / 2m ago / 3h ago)
- RSVP form — name input, I'm in / Can't make it buttons, "Remind me" soft text link
- Success screen on submit — personalized emoji + message, status in color, "reopen this link" copy

All API calls replaced the previous Claude artifact storage with real fetch calls. Loading and error states handled. Lucide icons used throughout (matching the Figma icon library).

---

### Key Implementation Decisions

**Activity feed instead of static roster**
Implemented as a reverse-chronological list of the last 8 responses with `timeAgo()` helper. Matches the Figma design and the group chat metaphor — live, timestamped, personal.

**Upsert-aware success screen**
No session memory, no accounts. The success screen explicitly tells participants how to change their RSVP: "Changed your plans? Just reopen this link and resubmit your name." The backend's upsert handles the rest.

**Status colors from design tokens**
Tailwind config uses the same token names from the Figma Variables collection (`status-in`, `status-out`, `status-remind`) — zero translation cost between design and code.

---

### What's Next

- MAT-55 — Deploy backend to Fly.io, frontend to Vercel

---

*Stack: React 19 · TypeScript · Vite · Tailwind CSS · Lucide React*
*Tools: VS Code · Linear*

---

## Session 05 — May 19, 2026

---

### What We Built

Full production deployment — Go backend on Fly.io, React frontend on Vercel. End-to-end flow live at `showup-frontend-eight.vercel.app`.

---

### Issues Closed

**MAT-55 — Deploy Fly.io (backend) + Vercel (frontend)**

**Backend — Fly.io**

- Wrote multi-stage Dockerfile (golang:alpine builder → alpine runtime, ~10MB final image)
- Created `fly.toml` with region `ewr` (Secaucus, NJ — closest to home base), 256MB shared VM, auto-stop/start enabled
- Provisioned Fly Postgres with scale-to-zero on the same region
- `fly postgres attach` automatically injected `DATABASE_URL` as a secret
- `FRONTEND_URL` secret set for CORS allowlist

**Frontend — Vercel**

- Connected `mattlau95/showup-frontend` GitHub repo
- Added `vercel.json` rewrite (`/(.*) → /index.html`) for client-side routing
- API URL switched to hostname-based routing after env var issues

---

### Things That Tripped Us Up

**Schema applied to wrong database**
`fly postgres connect -a showup-db` opens a session in the default `postgres` database. `fly postgres attach` creates a separate `showup_backend` database for the app. Tables had to be recreated in `showup_backend` after the 500 errors surfaced in production.

**Fly.io trial limit**
Free trial machines stop after 5 minutes without a credit card on file — added a card to unlock full operation.

**`VITE_API_URL` not picked up by Vercel**
A blank env var value set during initial project creation was being passed to the build, overriding the `.env.production` file (Vercel shell env vars take precedence over `.env` files). Fixed by switching to hostname-based API routing: `window.location.hostname === 'localhost' ? 'http://localhost:8080' : 'https://showup-backend.fly.dev'`.

**Pre-built `dist` folder in git**
The committed `dist` folder was being served by Vercel as static files instead of triggering a fresh build. Removed with `git rm -r --cached dist`.

---

### What's Next

- MAT-56 — "Remind Me" notification system (Phase 2)
- MAT-57 — Name input hint for common names
- MAT-54 — OG preview card implementation (Satori)

---

### Reflection

The deployment session had more friction than expected — all of it environmental rather than code. The schema/database mismatch is a Fly.io gotcha worth documenting: `fly postgres connect` and `fly postgres attach` operate on different databases by default. The VITE_ env var precedence issue is a Vite + Vercel combination problem that `.env.production` alone won't solve if the dashboard has a blank override. Both are worth remembering for the next deploy.

The app is live. Milestone from the ticket: drop a showup.gg link in the volleyball group chat.

---

*Stack: Go 1.26.3 · PostgreSQL 18 · Fly.io · React 19 · Vite · Vercel*
*Tools: flyctl · Docker · Linear*

---

## Session 06 — May 19, 2026

---

### What We Built

Pixel-accurate implementation of the full Figma design across both app pages, using the Figma MCP integration to pull design data directly into Claude Code. CSS design system updated to match Figma tokens. Redeployed to Vercel production.

---

### Issues Closed

**MAT-68 — Design implementation: Figma MCP + full UI redesign (RSVP + Create pages)**

---

### Figma MCP Integration

Connected Claude Code to the Figma file via MCP (`get_figma_data`). Instead of manually inspecting the Figma file, the tool returned the full node tree — layout modes, gap/padding values, fill colors (hex + rgba), stroke styles, border-radius, text styles, and component variants — as structured data.

The first fetch (node `3-51`, the Default frame) returned only one of three RSVP frames. Fetching the full file revealed the complete canvas structure: three RSVP frames (Default, Selected, Success), two Event Creation frames (Form, Share Link), OG Card, and all component sets with their variants. The second pass caught everything the first missed.

---

### RSVP Page (node 3-51) — All Three Frames

**Default state**
- Background `#0F172A`, padding `24px 16px`, section gap `24px`
- Title: Inter Bold 28px `#F8FAFC`
- Event details: Inter Bold 13px `#94A3B8` (not medium — bold, caught from `style_OWJN91`)
- Attending count: "17" Bold 24px + "attending" Regular 24px, both white — split weight within same line (Figma uses a text style override `ts1` on "attending")
- Activity feed box: `#273548` bg, `border-radius 8px`, `padding 16px 16px 0px` — content intentionally clips at bottom edge to imply more below
- "See All" link: centered (not left-aligned — the `layout_LA7V2A` has `justifyContent: center`)
- `flex-1` spacer pushes RSVP form toward bottom of screen, matching Figma's auto layout spacer
- Name input: `#273548` bg, `rgba(255,255,255,0.08)` border, 12px radius, 16px padding
- RSVP buttons: 50/50 split, `padding 20px 32px`, 32px emoji, 20px label text, column layout
- Submit disabled: `#273548` bg, `rgba(255,255,255,0.08)` border, `#94A3B8` text

**Selected state** (from `Status=In, State=Selected` and `Status=Out, State=Selected` component variants)
- "I'm in" selected: `#DCFCE7` bg, `#16A34A` border, `#22C55E` green text
- "Can't make it" selected: `#FEE2E2` bg, `#DC2626` border, `#EF4444` red text
- 😔 emoji unselected: dimmed — Figma fills it `#94A3B8` while ✅ is `#F8FAFC` white
- Submit active: `#F59E0B` amber bg, `#F8FAFC` white border + text (from `Property 1=Active` variant)

**Success state** (from frame `5:376`)
- Emoji: 64px (`style_GHPE1H` — easy to miss, looks like 48px at design zoom)
- Heading: Inter SemiBold 24px (`style_GZTV1D` — not bold)
- Status line: Regular 16px in status color
- Help text: `#64748B` (`fill_7HPJMY` — slate-500, distinct from the `#94A3B8` muted grey used elsewhere)
- "← View who's coming": white `#F8FAFC` underlined text link (not muted)

---

### Event Creation Page (node 2-50) — Both Frames

**Frame 1 — Form state**
- Padding: `48px 16px` (taller than RSVP's `24px` — different `layout_NBJE2O`)
- Header: Inter SemiBold 28px (`style_UNBNPU`)
- Labels: Inter Regular 12px, white `#FFFFFF` (`fill_JHSOHC`) — not muted, easy to miss
- 4 fields: Event Name, Date, Time, Location — same input component as RSVP name field
- Submit: `#F59E0B` amber, white border + text, 20px Regular — Active state from creation

**Frame 2 — Success/share state**
- Header and form fields: `opacity-[0.45]`, `pointer-events-none` — only those two groups, not the submit button
- Submit stays full opacity, relabels to "Edit Event Details"
- Share link card: `#1E293B` bg (`fill_ROF6PM`), `border-radius 16px`, `padding 24px`
  - "Your shareable link": SemiBold 11px muted, centered (`style_NFZW7T`)
  - URL: SemiBold 16px amber `#F59E0B`, centered
  - Copy row: clipboard SVG icon (20×20) + "tap to copy" SemiBold 16px muted

---

### CSS Design System Updates

Updated `index.css` `@theme` to match Figma token values:

| Token | Old value | New value (Figma) |
|---|---|---|
| `bg-base` | `#0a0a0f` | `#0F172A` |
| `bg-surface` | `#13131a` | `#273548` |
| `bg-elevated` | `#1c1c26` | `#1E293B` |
| `text-primary` | `#f3f4f6` | `#F8FAFC` |
| `text-secondary` / `text-muted` | `#9ca3af` / `#6b7280` | both `#94A3B8` |
| `text-disabled` | `#374151` | `#64748B` |

Added Inter font via Google Fonts in `index.html`. Set as primary font-family in `body`.

---

### Things That Tripped Us Up

**Fetching only one frame**
The initial fetch used node `3-51` (the Default frame URL). The full component data — including Selected and Success state frames, and all component set variants — only appeared when fetching the entire file without a node ID. Always fetch the full file first to see the canvas structure, then drill into specific nodes.

**😔 vs ✅ default color**
In the Default state, ✅ is `#F8FAFC` (white) but 😔 is `#94A3B8` (muted). Easy to miss because both look like "unselected" states. The distinction comes from `fill_8PPWJH` vs `fill_N8UKYW` in the component definition.

**Submit button active color**
The first implementation used white background for the active submit state. The Figma `Property 1=Active` variant uses `fill_NXPG1D` = `#F59E0B` amber — the same accent color used elsewhere in the design. Caught by fetching the full file and reading all component variants.

**`#64748B` vs `#94A3B8`**
Two distinct grey values in the design. `#94A3B8` (slate-400) is the general muted color. `#64748B` (slate-500) appears specifically on the success screen help text (`fill_7HPJMY`). They look nearly identical at design zoom but differ enough to matter in implementation.

---

### Deployed

Redeployed to Vercel production after all changes: `showup-frontend-eight.vercel.app`

---

### What's Next

- MAT-56 — "Remind Me" notification system (Phase 2)
- MAT-57 — Name input hint for common names
- MAT-54 — OG preview card implementation (Satori)

---

*Stack: React 19 · TypeScript · Vite · Tailwind CSS v4 · Inter · Vercel*
*Tools: Claude Code · Figma MCP · Linear*

---

## Session 07 — May 19, 2026

---

### What We Built

Five bugs caught on first real-device test, all fixed and redeployed. Backend gained a new `PATCH /events/:slug` endpoint.

---

### Issues Closed

**MAT-69 — Bug fixes: responsiveness, edit flow, timezone, icon visibility, desktop spacer**

---

### Bugs Fixed

**1. Create Event page not responsive**
`py-12` (48px top/bottom) with no mobile reduction meant the 4-field form + share card overflowed on short screens. Fixed: `py-6 sm:py-12`, tighter form gaps, `overflow-y-auto`.

**2. Edit Event Details was creating a new event instead of updating**
"Edit Event Details" was calling `setSlug(null)`, which cleared the slug and caused the next submit to hit `POST /events` — generating a brand new link and orphaning the old one. This is a meaningful product bug: any RSVPs already collected against the original link would be disconnected from the new one.

Fix: added `isEditing` boolean state. "Edit Event Details" sets `isEditing = true` (keeps slug, re-enables form). Button relabels to "Save Changes ->", which calls a new `PATCH /events/:slug` endpoint. The link never changes; the event is updated in place.

Backend changes:
- Added `UpdateEvent` handler: `PATCH /events/{slug}` — updates `title`, `location`, `event_date` in place with a single `UPDATE … RETURNING` query
- Added `PATCH` to CORS allowed methods in `main.go`

**3. Date/time picker icons invisible on dark background**
Native `<input type="date">` and `<input type="time">` render their calendar and clock icons using the OS/browser default color scheme, which is light-on-white by default. On the dark surface (`#273548`) these were nearly invisible.

Fix: `[color-scheme:dark]` on all inputs. This CSS property tells the browser to render all native UI chrome for that element (icons, spinners, dropdowns, scrollbars) in a light/white style. Applied to every input for consistency, not just date/time.

**4. Time displayed wrong on RSVP page (off by UTC offset)**
User entered 6:30 PM; RSVP page showed 2:30 PM — a 4-hour offset matching EST (UTC−4).

Root cause: the datetime was built as a raw string `${date}T${time}` (e.g. `2026-05-17T18:30`) and sent directly to the Go backend. Go's `time.Parse` treated it as UTC and stored `2026-05-17T18:30:00Z`. The RSVP page then called `new Date('2026-05-17T18:30:00Z').toLocaleTimeString()`, which correctly converted UTC to local time — giving 2:30 PM EST instead of the intended 6:30 PM.

Fix: `new Date(\`${date}T${time}\`).toISOString()` before sending. The JS `Date` constructor parses a no-timezone string as **local time**, so `.toISOString()` produces the correct UTC equivalent (`2026-05-17T22:30:00.000Z` for EST). Backend stores the right instant; RSVP page displays the right local time. Also extracted into a `buildEventDate()` helper shared by create and update.

**5. Spacer stretched into a void on desktop**
Both pages used `<div className="flex-1" />` to push the RSVP form / share card toward the bottom of the screen — matching the Figma auto layout spacer designed for a 393×853 iPhone frame. On a desktop browser with a tall viewport, the spacer stretched hundreds of pixels, creating a visually broken layout.

Fix: `max-h-12` caps the spacer at 48px. Mobile devices still get the thumb-zone separation; desktop gets a reasonable gap.

---

### Reflection

The timezone bug is a classic — and easy to miss in local testing if your machine happens to be UTC. The `${date}T${time}` pattern looks correct until you realize the backend treats the string as UTC. The rule of thumb: always convert local datetime input to an explicit UTC instant (`.toISOString()`) before sending to any backend that stores timestamps in UTC.

The edit flow bug was a product-level issue disguised as a UI bug. A new link would have meant any existing RSVPs were silently disconnected. Worth catching before anyone actually shares a link.

---

*Stack: Go 1.26.3 · React 19 · TypeScript · Vite · Tailwind CSS v4 · Fly.io · Vercel*
*Tools: Claude Code · Linear*

---

## Session 08 — May 24, 2026

---

### What We Did

Designed the Claude API integration for event creation — decisions made, issues filed, no code written yet. This session was product and UX design work.

---

### Feature Designed

**MAT-80 — Claude API: Natural language event creation**

Adding Claude API to the event creation flow. Instead of a traditional four-field form, the organizer types a single natural-language description and Claude parses it into structured fields shown in an editable preview before the event is created.

The portfolio framing: not a chatbot, not a wrapper. One API call, one gesture, immediate value — and easy to explain in an interview.

---

### Key Design Decisions

**One API call on submit, not streaming**
Email-style Smart Compose (streaming tokens on each keystroke) was considered and ruled out. Requires streaming calls on every keystroke pause — high latency, high cost, poor complexity-to-payoff for a 10–15 word input. Submit-and-parse is fast enough to feel instant. Autocomplete revisited in Phase 2 only for date/time normalization hints (JS-only, no API needed).

**Missing field handling — the B+C approach**
Three options considered:

- A) Leave missing fields empty and highlighted — user fills manually
- B) Always show editable preview — preview is the safety net, not a separate error state
- C) Smart defaults for null fields — missing time → 12:00 PM, missing date → next Saturday, missing location → TBD

Decision: B + C combined. Preview always renders. Missing fields get smart defaults with an amber visual indicator so the organizer sees what was assumed vs. inferred. A well-formed input is still one gesture; incomplete input is still graceful.

Back-and-forth clarification loop explicitly ruled out — the moment it asks questions, it feels like a form again. Breaks the "reaction energy, not form energy" principle.

**Input copy and placeholder strategy**
Label (persistent): "Describe your event" / "Include date, time, and location — Claude will handle the rest."

Placeholder examples that rotate:
- `Volleyball Saturday 2pm · Venice Beach Court 4`
- `Soccer Sunday 10am · Riverside Park`
- `Basketball this Friday at 6 · 24 Hour Fitness`

Goal: teach all four fields implicitly through example, not by listing requirements.

**Manual entry escape hatch**
Small secondary link below the input: "Fill in manually instead →". Reveals the standard four-field form. Signals AI input is optional cleverness, not a required hoop — and protects against misparse edge cases.

---

### Prompt Design (Planned)

System prompt instructs Claude to return JSON only — no preamble, no markdown fences:

```json
{
  "title": "Pickup Volleyball",
  "date": "2026-05-30",
  "time": "2:00 PM",
  "location": "Venice Beach Court 4"
}
```

Any field that cannot be inferred returns `null`. Frontend applies smart defaults for nulls before rendering the preview.

---

### Issues Created

| Issue | Title | Priority |
|---|---|---|
| MAT-80 | Claude API — Natural language event creation with parsed preview | High |
| MAT-81 | Claude API — Input field copy, placeholder examples & manual entry fallback | Medium |
| MAT-82 | Claude API — Missing field handling: smart defaults + editable preview (B+C) | Medium |

MAT-81 and MAT-82 are sub-issues of MAT-80.

---

### What's Next

- MAT-81 — Finalize input copy and implement manual entry toggle
- MAT-82 — Implement parsed preview card with smart default logic
- MAT-80 — Wire Claude API call through serverless function (key security), integrate end-to-end

---

*Tools: Linear · Claude*

---

---

## Session 09 — May 24, 2026

---

### What We Did

Product launched with real users, domain purchased and configured, product renamed from Showup.gg to Ollae.

---

### Launch Milestone

The app went live and was shared with the volleyball group. First session results:
- **8 RSVPs** received
- One participant RSVPed on behalf of their group (~8 players from Edison) — an unanticipated group RSVP behavior that surfaced immediately in real usage. Good case study material: real usage revealed an edge case within hours of launch.

---

### UX Issues Identified from Real Usage

**Link trust problem**
`showup-frontend-eight.vercel.app/events/fskfnxws` reads as a phishing link. Real users in a group chat have no reason to trust a random Vercel subdomain. Identified as the single highest-priority UX problem post-launch.

**Organizer edit access**
No way to return to edit an event after closing the browser. No accounts, no session memory — once the create page is gone, edit access is gone. Needs a solution before the app is used regularly.

**Platform admin visibility**
No way to see all events, clean up test data, or oversee the platform as the developer/owner.

---

### Issues Created

| Issue | Title | Priority |
|---|---|---|
| MAT-70 | Domain — purchase and configure showup.gg | Urgent |
| MAT-71 | OG card — implement /og/:slug endpoint + og meta tags | Urgent |
| MAT-72 | Organizer — admin token system for event editing | High |
| MAT-73 | Admin dashboard — platform-wide event and response management | High |
| MAT-75 | Rebrand — Showup.gg → Ollae + monorepo consolidation | High |

---

### Product Renamed: Showup.gg → Ollae

**Why the name changed**
USPTO search revealed "SHOWUP" is pending trademark registration by BW Events LLC in Class 009, 035, and 042 — directly covering downloadable mobile app software and SaaS. Too close to ignore.

**Naming process**
40+ names researched across English, Japanese, Korean, and Chinese/Taiwanese. Constraints that emerged during the process:
- Must be phonetically unambiguous on first read
- Action energy — not static or corporate
- Works for all group activities (sports, karaoke, board games) — product scope expanded beyond sports
- USPTO clean in Class 042
- Domain available

Notable eliminations: Quorum (041), Lockin (taken), Rally (taken), Pullup (041 — "arranging and hosting social events"), Turnout (035), Encore (taken), Reup (taken), Whosin (reads as "hoo-sin").

**Final name: Ollae (올래)**
Korean word meaning "wanna come? / you in?" — the casual invite you send before every pickup game, karaoke night, or board game session. Exactly the question the product answers.

- USPTO search: no results
- Phonetically clean: "OH-lay"
- Works for any group activity
- Strong brand story: named after the Korean phrase for the core product interaction

**Domain purchased**
`ollae.app` purchased on Porkbun. `.com` unavailable. `.gg` available but more expensive — `.app` is the better fit given broader-than-sports use case.

---

### MAT-70 — Domain Configuration

Pointed `ollae.app` to Vercel:

1. Added `ollae.app` as custom domain in Vercel project settings
2. Vercel provided A record: `@ → 216.198.79.1`
3. In Porkbun DNS: deleted existing ALIAS record (conflicting), added A record
4. DNS propagated in under 5 minutes (Cloudflare-powered)
5. SSL certificate auto-generated by Vercel
6. CORS fix: `fly secrets set FRONTEND_URL=https://ollae.app` — no code change needed, backend reads allowed origins from env var
7. Full flow verified: create event, RSVP, activity feed — all working on ollae.app

**MAT-70 closed ✓**

---

### Devlog moved to GitHub

Devlog committed to `showup-backend/docs/` on GitHub. Accessible from any device, versioned alongside the code. Standard end-of-session workflow: update here, commit to docs/.

---

### Frontend Fix — Favicon

Installed the full favicon package from realfavicongenerator.net. Previously the app had only a placeholder SVG.

Files added to `public/`:
- `favicon.ico` — legacy browser fallback
- `favicon.svg` — modern SVG favicon
- `favicon-96x96.png` — PNG fallback
- `apple-touch-icon.png` — iOS home screen icon
- `web-app-manifest-192x192.png` + `web-app-manifest-512x512.png` — PWA icons
- `site.webmanifest` — PWA manifest (`name: ollae.app`, `theme_color: #0F172A`)

Updated `index.html` with the full tag set: ICO shortcut, SVG, PNG, Apple touch icon, and manifest link. Committed `17505a0`, pushed to Vercel.

---

### Frontend Fix — Dynamic Browser Tab Titles

Both pages were showing the static `<title>showup.gg</title>` from `index.html` regardless of which page was open. Small but noticeable polish issue.

- **CreatePage** — `useEffect` on mount sets title to `"Create Event · showup.gg"`
- **RSVPPage** — sets `document.title` to `"{event.title} · showup.gg"` once the event data loads

Committed in `showup-frontend` as `717c4c0` (tab titles) and `1679429` (base HTML cleanup: title tag, Inter font, Figma-matched color tokens).

---

### What's Next

**Priority order:**
1. MAT-75 — Rebrand + monorepo consolidation (rename repos, update codebase, Figma, Linear)
2. MAT-71 — OG card implementation (Satori endpoint + og meta tags)
3. MAT-72 — Organizer admin token system
4. MAT-73 — Admin dashboard
5. MAT-80 — Claude API natural language event creation

---

### Reflection

The naming process took most of this session — 40+ names across four languages before landing on Ollae. The constraint that eliminated the most candidates was phonetic clarity: contractions without punctuation (Whosin, Rollin) look like different words entirely when written out. The Korean angle came from a personal connection and produced the strongest result — a word that means exactly what the product does, in a language that has a natural casual register for exactly this social context.

The domain going live in under 30 minutes after purchase (Cloudflare DNS + Vercel's auto-SSL) was the smoothest part of the session. The CORS fix being a single env var update — no code change, no redeploy — validated the environment-driven config pattern Claude Code set up in Session 05.

ollae.app is live. Real product, real name, real domain.

---

*Tools: Porkbun · Vercel · Fly.io · USPTO · Linear*

---

## Session 10 — May 25, 2026

---

### What We Built

Favicon package and wordmark logo added to the app — both deployed to production on ollae.app.

---

### Favicon (realfavicongenerator.net)

![favicon](https://raw.githubusercontent.com/mattlau95/showup-frontend/master/public/favicon.svg)

Generated a complete favicon package via realfavicongenerator.net and wired it up in `index.html` with the full tag set:

```html
<link rel="icon" type="image/png" href="/favicon-96x96.png" sizes="96x96" />
<link rel="icon" type="image/svg+xml" href="/favicon.svg" />
<link rel="shortcut icon" href="/favicon.ico" />
<link rel="apple-touch-icon" sizes="180x180" href="/apple-touch-icon.png" />
<link rel="manifest" href="/site.webmanifest" />
```

Files added to `public/`: `favicon.svg`, `favicon.ico`, `favicon-96x96.png`, `apple-touch-icon.png`, `web-app-manifest-192x192.png`, `web-app-manifest-512x512.png`, `site.webmanifest` (name: `ollae.app`, theme: `#0F172A`, display: `standalone`).

---

### Logo

![ollae logo](https://raw.githubusercontent.com/mattlau95/showup-frontend/master/public/ollae-logo.svg)

Added the amber `#F59E0B` wordmark to the top of both pages as a home button. The SVG source used `fill="white"` — recolored to `#F59E0B` to match the app's accent color before copying into `public/`.

- **CreatePage** — logo sits above the "Create Event" header, links to `/create`
- **RSVPPage** — logo sits above the event info, links to `/create` (home button for participants)

Sized at `h-7` (28px) — readable on mobile without competing with page content. Includes 1x/2x/3x PNG variants in `public/` for future use.

Commits: `17505a0` (favicon), `1a73f99` (logo). Deployed to production — ollae.app.

---

*Tools: Claude Code · realfavicongenerator.net · Vercel*

---

## Session 11 — May 25, 2026

---

### What We Built

Shipped MAT-80 (Claude API natural language event creation), closed the Ollae rebrand (MAT-75), and made a round of UX polish to the RSVP and Create pages.

---

### MAT-80 — Claude API: Natural Language Event Creation

The headline feature. Organizer types a freeform description; Claude parses it into structured fields; an editable preview appears before the event is created.

**Backend — `POST /parse-event`**

New endpoint in `internal/parse.go`. Calls `claude-haiku-4-5-20251001` via the Anthropic Messages API (no SDK — plain `net/http`). System prompt includes today's date so relative references like "Saturday" resolve correctly. Returns JSON with `title`, `date` (YYYY-MM-DD), `time` (HH:MM 24hr), `location` — nulls for fields that couldn't be inferred.

API key stored as a Fly.io secret (`ANTHROPIC_API_KEY`), read into `EventHandlers.AnthropicKey` at startup.

Robustness additions after the first 500:
- Check `resp.StatusCode != 200` before parsing — log the raw Anthropic error body
- Strip markdown code fences if the model wraps its JSON response

**Frontend — NL flow in `CreatePage.tsx`**

Two-view state machine (`nl` | `form`):

- **NL view** — single textarea, "Create Event →" button, "Fill in manually instead" escape hatch. Enter key triggers parse.
- **Form view** — same editable fields, pre-filled from parse result. Fields the model filled get an amber border (`border-[#F59E0B]/70`) until the user edits them. "← Try a different description" link at the bottom.

Subtitle, button label, and "Looks good?" prompt all adapt depending on whether the user came through NL or manual:

| State | Subtitle | Button |
|---|---|---|
| NL parse | "Here's what we found — give it a look before creating." | "Yes! Create the Event →" |
| Manual | "Fill in the details below, get a shareable link." | "Create Event →" |

Date and time fields moved to the same row to reduce vertical space.

---

### UX Polish — RSVP Page

- **Removed hardcoded 🏐 emoji** from the event title — it appeared regardless of event type
- **Event meta condensed to one line** — date, time, and location now render as `📅 Saturday, May 30 · ⏰ 2:00 PM · 📍 Venice Beach Court 4` using an IIFE that builds an array of present fields and joins with ` · `
- **RSVP button labels** reduced from `text-xl` to `text-base` to prevent "Can't make it" wrapping to two lines

---

### MAT-75 — Rebrand Closed

Marked Done in Linear. Remaining cleanup: archived old `showup-frontend` and `showup-backend` GitHub repos. Linear team rename skipped (solo project).

---

### Infrastructure

- `site.webmanifest` — fixed `theme_color` and `background_color` missing `#` prefix (browser was ignoring both values)

---

*Tools: Claude Code · Anthropic Console · Fly.io · Vercel*

---

## Session 12 — May 26, 2026

---

### What We Built

A fully dynamic OG preview card served as a 1200×630 PNG — rendered server-side in Go, with iMessage support, pixel-perfect Figma implementation, Inter fonts, the actual ollae logo, and an AI-picked emoji per event. Also shipped the emoji field end-to-end: Claude picks it during event creation, it renders faded in the card background and as a circle badge top-right.

---

### Issues Closed

**MAT-71 — OG card: /og/:slug endpoint + og meta tags**

---

### How It Works

**Routing (iMessage problem)**

iMessage link previews are fetched from the device using a regular Safari UA — not a bot UA like `applebot`. UA-based conditional rewrites never trigger for iMessage.

Solution: route every `/events/:slug` through the Go backend's `og-preview` handler, which returns an HTML page with full `og:*` meta tags and an immediate JS redirect to `?_src=app`. Vercel's `has.query` rule catches `?_src=app` and serves `index.html` (the SPA). iMessage reads the og tags and never executes JS. Regular users hit the redirect and land on the React app.

**Image rendering**

Used `fogleman/gg` — a Go 2D graphics library. Ruled out Satori (JS/Node, incompatible with the Go backend) and headless Chromium (too heavy for a simple PNG).

Fonts embedded via `//go:embed`:
- `Inter-Bold.ttf` and `Inter-Medium.ttf` from Inter v4.0 (GitHub releases)
- `OpenMoji-black-glyf.ttf` for emoji (1.5 MB, monochrome glyf outlines — the only emoji font that works with Go's `opentype` package; Noto Emoji dropped their monochrome TTF in favor of COLRv1)

**Pixel-perfect Figma layout**

The Figma OG card uses `justify-content: space-between` across 4 sections: logo row, title+tagline, 72px spacer, meta+button. Replicated exactly:

```
gap = (534px innerHeight − totalSectionHeight) / 3
```

Each section's Y position computed dynamically so fonts at different sizes still distribute evenly.

**Optical text centering**

`fogleman/gg`'s `DrawStringAnchored` with `ay=0.5` centers using `fontHeight` (ascent + descent + line gap). This places the visual glyph above the optical center. Fixed with a `capCenter()` helper that reads `face.Metrics().CapHeight` and computes:

```go
baseline = boxY + (boxH + capH) / 2
```

Used for the RSVP button text. Same principle applied to the emoji circle badge using ascent/descent metrics.

**DPI gotcha**

`opentype.NewFace` with `DPI: 96` made fonts 33% larger than Figma's CSS px values. Fixed by using `DPI: 72` — at 72 DPI, 1pt = 1px, which matches Figma's font-size numbers exactly.

**Emoji field**

Added `emoji TEXT NOT NULL DEFAULT ''` to the `events` table. Two acquisition paths:
- **NL parse path**: added `"emoji"` to the Claude system prompt — same Haiku call, no extra cost. Frontend stores it and sends it with the create request.
- **Manual create path**: if `emoji` is absent from the create body, `CreateEvent` calls Claude separately to pick one based on the title (~1s extra latency, acceptable for a one-time operation).

OG card renders the emoji twice: as a 540px faded (17% opacity) background element on the right, and as a 48px glyph inside a 96×96 circle badge top-right.

**OG image URL**

Original implementation hardcoded `https://ollae-backend.fly.dev/og/:slug` in the `og:image` meta tag — an internal infra URL leaking into public meta tags. Fixed by adding a Vercel proxy rewrite:

```json
{ "source": "/og/:slug", "destination": "https://ollae-backend.fly.dev/og/:slug" }
```

All public-facing OG image URLs are now `https://ollae.app/og/:slug`.

---

### Things That Tripped Us Up

**Migration on the wrong database**

`fly postgres connect -a showup-db` connects to the `postgres` default database. The actual app database is `showup_backend`. Running `ALTER TABLE` without `-d showup_backend` silently succeeded — on the wrong database. The backend returned 500 on every create because the column didn't exist in the right schema.

Fix: always pass `-d showup_backend` explicitly. The cluster has three databases: `postgres` (default), `ollae_backend` (unused), `showup_backend` (live).

**Noto Emoji dropped their monochrome TTF**

The plan was `NotoEmoji-Regular.ttf` — ~500 KB, standard glyf outlines, compatible with Go's `opentype`. The file no longer exists in the repo; Google replaced it with COLRv1 and CBDT formats, neither of which Go can render. Switched to `OpenMoji-black-glyf.ttf` from OpenMoji v17.

**Timezone regression**

Session 07 fixed the RSVP page timezone bug by adding `new Date(...).toISOString()`. That same UTC conversion meant the OG card (running on a UTC server) displayed UTC time, not the user's local time. Fix: removed `toISOString()` and sent the local datetime string directly. The Fly.io server treats no-timezone strings as UTC, so the stored value matches what the user typed, and both the RSVP page and OG card display it correctly.

---

### Product Decisions Made

| Decision | Rationale |
|---|---|
| One emoji per event, AI-picked | Adds personality to the OG card without any UX burden on the creator. |
| Emoji picked at create time, not lazily | 1s latency on create is invisible; slow first OG render would be noticeable. |
| Monochrome emoji (OpenMoji Black) | At 17% opacity on dark slate, color vs. monochrome is indistinguishable. Circle badge at full opacity looks intentional. |
| OG image proxied through ollae.app | Internal fly.dev URLs in public meta tags are an implementation detail leaking into every iMessage preview. One rewrite rule, clean public surface. |

---

### Reflection

The iMessage routing problem is a good case study in assumption failure. UA-based filtering sounds reasonable until you learn iMessage fetches previews from the device, making every request look like a regular Safari browser. Any solution based on UA sniffing misses it entirely. The fix — routing all event URLs through an og-preview endpoint with a JS redirect bypass — is elegant once you understand the constraint.

The emoji field is small addition with outsized effect on shareability. An OG card with a relevant emoji reads as human and intentional. The single-call parse approach (one Claude request returns title, date, time, location, and emoji) keeps the cost at zero marginal per field — a useful pattern to remember when extending structured extraction.

---

*Stack: Go 1.26.3 · React 19 · TypeScript · Vite · Tailwind CSS v4 · Fly.io · Vercel · fogleman/gg · OpenMoji*
*Tools: Claude Code · Linear · Figma MCP*

---

## Session 13 — May 26, 2026

---

### What We Built

Debugged and fixed Facebook Messenger not showing OG card previews. Documented as MAT-87.

---

### Issues Closed

**MAT-87 — Facebook Messenger OG preview not showing**

iMessage worked; Facebook Messenger didn't. Took three root causes to explain it.

---

### Things That Tripped Us Up

**Stale Vercel serverless function still in git**

`frontend/api/og/[slug].tsx` — an old `@vercel/og` ImageResponse function from a previous OG implementation — was still tracked in git. Vercel's git integration deploys everything it sees, so that function was live at `/api/og/:slug`. Facebook's scraper picked up that URL from the og metadata and used it instead of the correct `/og/:slug` proxy. The Facebook Sharing Debugger showed `og:image: https://ollae.app/api/og/<slug>` — the dead giveaway.

Fix: `git rm frontend/api/og/[slug].tsx frontend/api/tsconfig.json frontend/middleware.ts`. Committed and pushed.

**`http-equiv="refresh"` vs `location.replace()`**

The og-preview endpoint originally used `<meta http-equiv="refresh" content="0;url=...?_src=app">` to redirect browsers to the SPA. Facebook's scraper follows meta refresh tags — it ended up at `?_src=app`, Vercel served `index.html`, no og tags. iMessage fetches using Safari which doesn't execute JavaScript but also doesn't follow meta refresh, so it hit the og-preview HTML directly and got the tags.

Fix: removed the meta refresh entirely. Kept only `<script>location.replace("...")</script>` — scrapers don't execute JS, so they read the og tags and stop; browsers get redirected to the SPA.

**HEAD requests return 405**

Facebook validates og:image URLs with a HEAD request before scraping the full image. The Vercel rewrite passes HEAD through to the Go backend, which serves the route with a regular handler — Go's `net/http` responds to HEAD automatically for any GET handler. This turned out not to need an explicit fix, but it's worth noting as a potential failure point when og:image endpoints are behind custom middleware.

---

### Product Decisions Made

| Decision | Rationale |
|---|---|
| Remove old API files entirely | Leaving dead Vercel functions in git is a deployment liability — they're silently live and can shadow new routes. Deleted, not just commented out. |
| JS redirect only (no meta refresh) | Meta refresh is followed by crawlers; JS redirect is not. Only one of these works for the og-preview use case. |

---

### Reflection

The iMessage vs Facebook Messenger split was a useful diagnostic. When one platform works and another doesn't, the difference is almost always scraper behavior, not the og tags themselves. Facebook's scraper is stricter — it follows redirects, validates image URLs with HEAD, and caches aggressively. The Facebook Sharing Debugger's "Scrape Again" button and the redirect chain it shows were the most useful debugging tool here.

The stale serverless function is a good reminder that Vercel deploys the git tree, not what you have locally. A file you haven't thought about in weeks can still be live in production.

---

*Stack: Go 1.26.3 · React 19 · TypeScript · Vite · Tailwind CSS v4 · Fly.io · Vercel · fogleman/gg · OpenMoji*
*Tools: Claude Code · Linear · Figma MCP*

---

## Session 14 — May 27, 2026

---

### What We Built

Three bug fixes caught from iPhone screenshots, and MAT-72 — a full admin token system that gives event organizers a persistent edit link without requiring accounts.

---

### Issues Closed

**MAT-72 — Organizer admin token system for event editing**

Previously, closing the create page meant losing all ability to edit the event. No accounts, no persistence. Fixed with a dual-link system:

- `POST /events` now generates a 24-char nanoid `admin_token` and returns it alongside the slug
- `PATCH /events/:slug` now requires `?admin=TOKEN` and validates with `WHERE slug = $1 AND admin_token = $2` — wrong or missing token returns 403. One query does both the auth check and the update.
- CreatePage shows two links after creation: the public share link and an edit link labeled "Save this to edit later — don't share"
- RSVPPage reads `?admin=TOKEN` via `useSearchParams`. When present, an "✏️ Edit event" button appears that expands an inline edit form pre-filled from the current event data. Save calls PATCH with the token; on success the event display updates in place.
- Vercel rewrite: added a `has: { type: query, key: admin }` rule so `?admin=TOKEN` URLs serve `index.html` directly instead of going through the og-preview handler (which would JS-redirect to `?_src=app`, dropping the token).
- Auto-scroll: after event creation the page smooth-scrolls the "Edit Event Details" button into view so both share cards are visible on mobile.

Database migration: `ALTER TABLE events ADD COLUMN IF NOT EXISTS admin_token TEXT NOT NULL DEFAULT ''`. Existing events get an empty token and are effectively locked from editing — acceptable since there was no persistent edit access before anyway.

---

### Bug Fixes

**Date/Time input overlap on mobile**

The Date and Time inputs in the Create Event form share a `flex` row with `flex-1` on each column. iOS native `<input type="date">` has a large intrinsic content width ("May 27, 2026") that won't shrink below its natural size in flexbox without `min-w-0`. The Time input was being pushed off the right edge of the screen.

Fix: `min-w-0` on both flex children. Standard Tailwind fix for this class of flex overflow.

**Event time shows wrong on RSVP page (timezone offset)**

User entered 10:00 PM; the RSVP page showed 6:00 PM — exactly 4 hours off (EDT, UTC−4).

Root cause: `buildEventDate()` sends `2026-05-27T22:00:00` (no timezone info). PostgreSQL's `TIMESTAMPTZ` column interprets the naive string as UTC and stores it that way. The backend returns `2026-05-27T22:00:00Z`. JavaScript's `new Date("...Z")` converts UTC to local time, so EDT viewers see 6:00 PM.

Fix: `parseAsWallClock()` in RSVPPage strips the `Z` suffix before constructing the `Date` object, so JS treats the stored value as the wall-clock time the organizer entered. `formatDate`, `formatTime`, and `hasTime` all use it. This is a display-layer fix — the semantic model is "store wall-clock time as UTC, display as entered."

**Facebook OG: HEAD requests returning 405**

Every route in the Go backend is registered with `r.Get(...)`. chi does not automatically handle HEAD for GET routes, so HEAD requests returned 405 Method Not Allowed. Facebook's scraper linter sends HEAD to validate both the page URL and the `og:image` URL before rendering a card.

Fix: `headToGet` middleware added to the chi router. It converts any incoming HEAD request to GET (mutating `r.Method`), wraps the `ResponseWriter` in a `noBodyWriter` that discards all `Write` calls, then passes through to the normal handler. The response headers (Content-Type, Cache-Control) are set correctly; the body is silently dropped. Also added `HEAD` to the CORS `AllowedMethods` list. Facebook's cache was flushed manually via the Sharing Debugger after deployment.

---

### Issues Created

**MAT-89 — RSVP +1 / bring guests support**

Users sometimes want to RSVP for a group without submitting separate entries. Proposed: a guest count selector (just me / +1 / +2 / +3) that appears after selecting "I'm in", with submit button text changing to "We're in! →" when guests > 0. Attending count would reflect total headcount. Parked in backlog.

---

### Things That Tripped Us Up

**TypeScript "used before declaration" in React components**

A `useEffect` referencing `slug` and `isEditing` was placed before those `useState` declarations in the component function. TypeScript's strict mode flags this as TS2448 (block-scoped variable used before declaration), even though the callback only runs after all declarations have executed. The local `tsc --noEmit` passed; the Vercel build caught it. Fix: move `useEffect` calls that reference state to after all `useState`/`useRef` declarations.

**`?admin=TOKEN` dropped by og-preview redirect**

The Vercel rewrite sends any `/events/:slug` request without `?_src=app` to the Go og-preview handler. That handler returns HTML with a JS `location.replace()` to `?_src=app`. When the organizer visits their admin link, the redirect fires and drops the token — the SPA loaded but `useSearchParams().get('admin')` returned null.

Fix: added a `has: { type: query, key: admin }` rewrite rule in `vercel.json` before the og-preview catch-all. Rewrite rule order matters — first match wins.

---

### Product Decisions Made

| Decision | Rationale |
|---|---|
| Empty token default for existing events | Simpler migration than generating tokens for all existing rows. Pre-MAT-72 events had no persistent edit access anyway — locking them is consistent with prior behavior. |
| Wall-clock time model (no timezone info) | ollae events are informal and local. Storing and displaying the time as entered is what organizers expect. Viewers in different timezones don't get automatic conversion — acceptable for this use case. |
| Token in query param, not header | The admin link needs to work by tapping a URL in a message thread. Headers aren't part of a URL. Query param is the only option for a link-based auth model. |
| `WHERE slug = $1 AND admin_token = $2` in UPDATE | Single query does auth check and update atomically. `sql.ErrNoRows` means either the slug doesn't exist or the token is wrong — no need to distinguish the two cases. |

---

### Reflection

The timezone bug is a recurring theme in this project. The root cause is always the same: `TIMESTAMPTZ` + naive string input + `new Date("...Z")` display = automatic UTC→local conversion that nobody asked for. The wall-clock model (strip the Z, display as entered) is the right answer for this class of app. If your app stores "the time someone said" rather than "a UTC moment in time", opt out of automatic timezone conversion at the display layer.

The admin token feature shows how much product surface you can cover without accounts. A 24-char opaque token in a URL is sufficient for casual private events — the organizer has it, no one else does, and it's as persistent as any link. Phase 2 can layer real accounts on top without schema changes.

---

*Stack: Go 1.26.3 · React 19 · TypeScript · Vite · Tailwind CSS v4 · Fly.io · Vercel · fogleman/gg · OpenMoji*
*Tools: Claude Code · Linear*
