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
