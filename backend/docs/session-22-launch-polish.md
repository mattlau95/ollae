# Session 22 — Sep 15, 2026 — Pre-launch audit + polish

**Goal:** get the repo and app ready for a Product Hunt launch and for recruiters reading the code for Design Engineer roles.

**Method:** a read-only audit first (repo legibility, code quality, WCAG 2.2 AA + UX checklist §1–10, launch readiness), with Lighthouse / axe / pa11y run against production. Then fixes in severity order — P0, P1, P2 — one commit per fix. 32 commits, none pushed or deployed yet.

**Baseline from the live site:** Lighthouse perf 91 / a11y 97 / best-practices 100; LCP 2.8s, CLS 0. axe on `/` found only landmark issues; on a dead event slug the response was a bare plain-text `not found` from the backend, never the app.

---

## P0 — blockers

| Commit | What | Why |
|---|---|---|
| `d3500a4` | Removed the hardcoded Postgres fallback URL in `main.go`; backend now fails fast without `DATABASE_URL` | A real-looking password was in source since the first commit. **Still in git history — rotate it.** |
| `0cb4323` | `--color-text-disabled` #64748B → #94A3B8 | Measured 3.8:1 on the base background; AA needs 4.5:1 for body text |
| `5298af2` | `sr-only` labels on the four placeholder-only inputs (RSVP name, reminder email, NL textarea, admin password) | Placeholders vanish on input; screen readers had no field name |
| `a37857f` | `htmlFor`/`id` on the eight visibly-labeled fields (create form, admin edit form) | Labels were siblings, not associated (WCAG 1.3.1 / 4.1.2) |
| `0c77619` | `og-preview` redirects browsers into the SPA on a missing event; crawlers still get 404 | Every `/events/:slug` is proxied through the OG handler first, so expired/mistyped links showed raw text instead of the app's error state |

## P1 — important

| Commit | What |
|---|---|
| `30d08cc` | `RescrapeEvent` moved under `/admin` and gated on `adminAuth` — it was the one admin action anyone could call |
| `2326fe7` | `NewDB` returns an error instead of `log.Fatalf` from inside the package; stray `fmt.Println` → `log` |
| `5c411cd` | Admin secret compared with `crypto/subtle.ConstantTimeCompare` |
| `57118a7` | README documents that organizer edit links carry a bearer token in the URL on purpose |
| `38b854f` + `463839f` | `npm run lint` passes (3 errors, 2 warnings). Second commit needed because the first tripped `set-state-in-effect`; admin data load is now a promise chain inside its effect |
| `3ec5b5d` | `showup-devlog.md` → `ollae-devlog.md`, retitled, pointer to Session 09 for the rename |
| `964eb2b` | API base URL centralized in `api.ts`; dead `.env.production` (still pointing at `showup-backend`) deleted |
| `15734d7` | Global unlayered `:focus-visible` outline — inputs had `outline-none` + a 25%-opacity border shift |
| `3b432bf` | Admin page `gray-500/600` text → `gray-400` (was ~3–3.7:1) |
| `bea763a` | `celebrate()` helper checks `prefers-reduced-motion` before confetti; dedupes the particle config |
| `bc95175` | RSVP page error state: "doesn't exist" vs "couldn't load", Try again, link to create, `role="status"` / `role="alert"` |
| `cf2aa90` | Admin icon buttons get `aria-label` + `aria-expanded`, 24px targets; checkboxes 20px |
| `6fa21a1` | README architecture tree + route list rewritten — half the backend (admin, OG, reminders, Claude parsing) was missing; names the model; explains share-link routing; lists optional env vars; links the devlog |
| `be098cb` | Create page tagline for cold visitors ("One link for group plans…"); Claude line reworded to say fields are editable |
| `ecaa66c` | Zero-RSVP empty state |
| `032df4a` | `Toast` component replaces all ten `window.alert()` calls; admin page no longer shows "Loading…" forever on a failed first request |
| `997701f` | `api()` wrapper with 15s `AbortSignal.timeout`; RSVP page shows "Waking up the server" after 3s. `/parse-event` left on plain `fetch` |

## P2 — polish

| Commit | What |
|---|---|
| `8bacc6e` | Dropped `lucide-react` (never imported) and `src/assets` scaffold files |
| `b0fd63c` | Root `.gitignore`; `.vercel/project.json` untracked |
| `0f41bac` | `autoComplete` on name/email |
| `b9aa02c` | "Creating…" → "Filling in the details…" during parsing |
| `47cd96f` | `JSONError` helper — every API handler fails with `{"error": …}` (OG endpoints and `parse.go` untouched) |
| `a5abe96` | First tests: `isCrawlerUA`, `truncate`, `escapeHTML`, `formatOGWhen`, `adminAuth` |
| `7456789` | gofmt `og.go` |
| `3915931` | Guest pills 44px, text-link buttons padded |
| `0957b3f` | README mentions `go test ./...` |

## Audit findings verified as already fine

OG/Messenger crawler fix is correct end to end (`vercel.json` UA-gated rewrites → `og.go` `isCrawlerUA` → `location.replace` to `?_src=app`). Claude key is server-side only; user input goes in as the user turn, not concatenated into the system prompt; 15s client timeout; malformed/markdown-fenced replies handled. All SQL parameterized. AI-parsed fields are amber-flagged and editable. Form state survives failed submits. Semantic buttons/headings throughout.

## Not done — needs a decision

- **Crawler bounce redesign (P1 🔨):** every real visitor pays backend HTML → JS redirect → SPA before seeing the RSVP page. Fix is a UA-keyed rewrite in `vercel.json`, but it touches the FB preview path and can only be verified by re-scraping after deploy.
- **Claude hardening:** retry/backoff on 429/5xx in `parse.go`; rate limit on `/parse-event`. Stopped short because it's the Claude integration.
- **`/parse-event` client timeout** — the one fetch not on `api()`.
- Devlog Sessions 18/19 are in reverse order in the file (both dated Jun 3).
- No frontend tests (Vitest not set up).

## Follow-up (same day)

Remaining work filed as Linear issues MAT-705 → MAT-710 in project "Ollae". Then:

| Commit | What |
|---|---|
| `4f09d8f` | Devlog Sessions 18/19 put in order (MAT-710) |
| `7f36db6` | `parse.go`: Anthropic call retries on transport error / 429 / 5xx with backoff, honors `Retry-After`, 25s deadline; surviving 429 → 503. `main.go`: `httprate` on `/parse-event`, 10/min per client (keyed on `Fly-Client-IP`) + 120/min global. Tests for the retry paths. (MAT-709) |
| `d00cb80` | `/parse-event` on `api()` with a 30s timeout; 429/503 → "busy" toast (MAT-709) |

MAT-706 verified without changes: `api.ts` default = `ollae-backend.fly.dev`, `/health` 200; the old host is dead; nothing ever read `VITE_API_URL`.

MAT-708 browser QA: 33/33 Playwright checks against the real frontend + a stub API, screenshots reviewed. Finding: the empty-name RSVP toast is unreachable because Submit is disabled until a name is entered. Real-backend smoke remains part of the deploy (MAT-707).

Still yours: MAT-705 (rotate password), MAT-707 (deploy), and the crawler-bounce redesign (not filed).

## Before deploying

1. Rotate the Postgres password everywhere it was used.
2. Click through: create an event (tagline, parse failure → toast), RSVP with empty name (toast + focus), `/events/nope` (new error page), Tab through the create form (focus outline). Nothing in this session was run in a browser — only `tsc`, `eslint`, `vite build`, `go build/vet/test`.
3. `fly deploy` for the backend; frontend ships on push.
4. Optionally squash `38b854f` + `463839f`.
