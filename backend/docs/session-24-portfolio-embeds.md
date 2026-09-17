# Session 24 — Sep 16–17, 2026 — Portfolio embeds (MAT-720)

The guestbook and the create demo can now be framed by matthewclau.com. Three deploys went out: a privacy fix, reminder groundwork, and the embed feature with its abuse limits. All are verified against production.

**Backend:** Fly.io `ollae-backend` (release v32) · **Frontend:** Vercel `ollae.app` · **Branch:** `master` at `c162ea0`

---

## At a glance

| | Status |
|---|---|
| Both embeds | **Live.** One CSP header per response on every route, allowlist only where intended. |
| Privacy fix | **Live.** Reminder emails no longer appear in public event JSON. |
| Append-only guestbook | **On.** Your RSVP stays; nobody can resubmit a name to change it. |
| Guestbook reminders | **Off.** Remind me is recorded; no email is taken or sent. |
| Blocked words | **30 in production**, managed in `/admin` (collapsed and masked). None in the repo or its history. |
| iPhone check | **To do**, once the portfolio page with the iframes is live. |

---

## URLs for the portfolio

- **Guestbook:** `https://ollae.app/events/wssrfd7v?_src=app&embed=1`
- **Create demo:** `https://ollae.app/create?embed=1`

On each iframe, set `allow="clipboard-write"` so "tap to copy" works. Don't set `referrerpolicy="no-referrer"`: Firefox needs the referrer to identify the parent page.

Framing is allowed only from `https://www.matthewclau.com`, `https://matthewclau.com` and `http://localhost:4321`, and only when `embed=1` is present and no `admin` token is.

## Message protocol

Messages are only exchanged with an allowlisted parent origin, never `'*'`.

| Message | Direction | Screens | When |
|---|---|---|---|
| `{ type: 'ollae:height', height }` | to parent | both | On load and whenever the content wrapper changes size (ResizeObserver) |
| `{ type: 'ollae:ready' }` | to parent | create | Once the prefill listener is attached |
| `{ type: 'ollae:input', text }` | to parent | create | Every change to the description, including after a prefill |
| `{ type: 'ollae:created' }` | to parent | create | After an event is created |
| `{ type: 'ollae:prefill', text }` | from parent | create | Fills the box without submitting; accepted only from `window.parent` at an allowlisted origin |

---

## What shipped

### A · Privacy fix — deployed and pushed

- Public `GET /events/:slug` and the RSVP response used to include `notify_via`, so anyone with a link could read every Remind me email. Both now return a type without it; `/admin` still has it.
- Fly had been running the backend from `14f60c0` (Jun 3). This deploy also shipped 40 finished commits that had never reached Fly: the June OG/Facebook fixes and the Session 22 launch polish.
- Sequenced after MAT-705 (Postgres password rotation, Session 23).

```
d6c4146 fix: stop exposing reminder emails on public event endpoints
9609294 docs: add Session 23 devlog — MAT-705 Postgres password rotation
```

### B · Groundwork — deployed and pushed

- `schema.sql` synced with production (`showup_backend`). The old file didn't load, and was missing `guests`, `reminded_at` and `settings`.
- Reminders are claimed atomically (`SKIP LOCKED`) before sending, so the loops on two machines and the cron job can't send twice. A batch that fails is released for retry.
- Sends use Resend's batch endpoint, capped at the Free plan's daily quota, paced under its request rate, with a 15 s timeout. Names and titles are HTML-escaped in the email.
- Stored emails are cleared once an event is more than a day past.

```
1fe3b8f docs: sync schema.sql with the production database
38c90c5 fix: claim reminders atomically and batch them within Resend's limits
55bad5a feat: clear reminder emails a day after the event
```

### C · Embeds, guestbook, demo events, abuse limits — deployed and pushed

- **Framing:** CSP `frame-ancestors` on both the Vercel app and the Go SSR page; `'none'` everywhere else. The SSR redirect keeps every query parameter, including `embed=1`.
- **Embed mode:** content height, height messages, keeps the previous height when a screen changes, links open in new tabs, toasts render inline.
- **Guestbook:** response count, 8 most recent with "Show all N", per-event append-only with a friendly duplicate-name error, organizer delete from the edit link, noindex.
- **Create demo:** the messages above, demo placeholder, OG card on the share screen, events flagged `is_demo` (no Remind me, noindex, 3-day note, deleted by an hourly cleanup).
- **Abuse limits:** per-IP rate limits, a daily Claude cap in Postgres, server-side validation, and blocked words managed in `/admin` (details below).
- **Date default:** a time with no date defaults to today, or tomorrow once that time has passed, with its own hint.
- **Admin:** append-only toggle and a demo badge on `/admin`.
- Append-only enabled on `wssrfd7v` after deploy.

```
c55f602 feat: frame the guestbook and create screens for the portfolio, with abuse limits
9942f28 feat: embed mode for the guestbook and create demo
2925172 fix: leave the date empty when a description names no day
```

---

## Limits and validation

| Endpoint | Per client | Also |
|---|---|---|
| Parse (Claude) | 5 / min · 30 / hour | 120 / min for everyone; 300 Claude calls per UTC day (`CLAUDE_DAILY_CAP`), counted in Postgres |
| Create event | 5 / min · 20 / hour | The emoji call on manual create counts toward the daily cap |
| RSVP | 10 / min · 60 / hour | — |

| Field | Rule |
|---|---|
| Name | Trimmed, up to 40 characters, blocklist |
| Event title | Up to 80 characters, blocklist |
| Location | Up to 120 characters, blocklist |
| Guests | 0–20 on every event (was 99) |
| Reminder email | Up to 254 characters, basic format check |
| Description to parse | Up to 500 characters |
| Emoji | Exactly one emoji; otherwise 🎉 from the parser, or none on create |
| Parsed output | Title and location truncated; dates and times that don't match the format become empty |

Rate limits live in memory, so each Fly machine counts separately (2 machines, one usually stopped). The daily Claude cap is shared.

```
6b2a5f9 fix: keep blocked words out of the repo, and turn reminders off on the guestbook
cd4c98a feat: board game night as the create demo's example
c162ea0 feat: hide blocked words on /admin until shown, and mask them
```

History was rewritten to remove blocked terms from the commit that first added the list, and force-pushed. Hashes in this document are the rewritten ones. GitHub still serves the old commit by exact hash until a Support request removes it.

**Follow-up:** blocked words now live only in a `blocked_terms` table, edited in `/admin` (collapsed and masked) and reloaded on every machine within 15 seconds. None are in the repo, and an empty list lets everything through. Production was seeded with the 30 original terms. Matching is by whole word, after lowercasing, undoing character swaps and joining spelled-out letters. The guestbook has `reminders_off`: Remind me is still an answer, but no email is taken or sent.

---

## Answers to your questions

**Can the embed skip the SSR hop?**
Yes. With `?_src=app`, Vercel serves the app directly. The SSR page still sends the right header and keeps `embed=1`, because the app removes `_src` from the address bar and a reload goes through it.

**Which screens get a full page load inside the frame?**
Only the two starting pages, plus the SSR page if the guestbook frame is reloaded. RSVP → success, Show all, and create → share are client-side, and every link that would navigate opens a new tab instead.

**How is the parent origin found?**
`location.ancestorOrigins` in Chromium and Safari; `document.referrer` in Firefox (valid because the embed URL skips the SSR page); an allowlisted `ollae:prefill` also counts. It must match the allowlist, or nothing is posted.

**Are normal event pages indexed?**
They're indexable: no robots.txt, meta robots tag or `X-Robots-Tag`. Only Search Console can say whether any are actually indexed. The guestbook and demo events are now noindex.

**Do API requests pass through a proxy that changes the IP?**
No. The browser calls `ollae-backend.fly.dev` directly, so `Fly-Client-IP` is the visitor, and Fly overwrites any value a client sends. Only `/og` and the SSR page go through Vercel, and neither is rate-limited. IPv6 is grouped by /64.

**What does the example parse to?**
Title "Ollae Demo", location "Alexander Library", 12:30 PM, date left empty so the app's default applies.

**What date after 12:30 PM?**
Tomorrow, with the hint "No date given, and that time has passed today, so we picked tomorrow." Before 12:30 it picks today.

**Which emoji across five runs?**
With the final prompt: 🎯 🎯 🎯 🎤 🎥. Before the prompt fix: 🎬 🎯 🎥 🎯 💻. Never 📚.

**Reminder batches and Resend limits?**
Free plan: 100 emails a day, 3,000 a month, 10 requests a second, 100 per batch, every recipient counts. Reminders stop at the daily quota and log what wasn't sent. A large guestbook would need a paid plan.

**WebKit?**
Not run, at your request. You'll check both embeds on an iPhone after the portfolio page is live.

---

## Verification

### Production headers (ollae.app)

| Request | CSP | Other |
|---|---|---|
| `/events/wssrfd7v?_src=app&embed=1` | allowlist | noindex |
| `/events/wssrfd7v?embed=1` (SSR page) | allowlist | noindex |
| `/create?embed=1` | allowlist | — |
| Another event with `embed=1` | `'none'` | — |
| Guestbook with `admin` and `embed=1` | `'none'` | noindex |
| Guestbook without `embed`; `/create`; `/` | `'none'` | — |

Every response had exactly one `Content-Security-Policy` line with one `frame-ancestors`, and no `X-Frame-Options`. On the SSR route, Vercel's header replaces the one from Fly.

### Browser (Chromium)

- Local embed checks: 31 of 32 passed. The only failure was Chrome logging the deliberate duplicate-name 409.
- Date default with a stubbed parser: 5 of 5 passed.
- Real parser end to end: 4 of 4, including an off-script event flagged `is_demo`.
- Production framed from `localhost:4321`: both frames load, heights arrive, `ready` and prefill work, no frame-blocked errors.

### Backend

- 33 Go tests pass, including new ones for validation, the blocklist, rate limits, the daily cap, append-only, organizer delete, cleanup, reminders and framing.
- Mutation checks: removing `SKIP LOCKED`, adding a second CSP header, and disabling append-only each made their tests fail.
- Live locally: the daily cap, both rate limits, and cleanup deleting an old demo while keeping a real event of the same age.

---

## Found and fixed along the way

- **Privacy:** public event JSON exposed reminder emails (deploy A).
- **Deploy state:** Fly was 40 commits behind `master`, including OG fixes Vercel already had.
- **Reliability:** reminders could double-send across machines and had no HTTP timeout.
- **Security:** visitor names and titles went into reminder email HTML unescaped.
- **Parsing:** Claude assumed today's date for a time-only description, which would have created past events after 12:30.
- **Parsing:** the parser used the server's UTC date; it now uses the visitor's local date.
- **Tooling:** `vercel dev` ignores `has` conditions and sent no CSP headers, so header rules were verified on a preview deployment instead.

---

## Still open

- **You:** iPhone check in Safari once the portfolio embeds are live (checklist below).
- **Watch:** Resend's Free plan allows 100 emails a day. Reminders beyond that are logged, not sent.
- **Optional:** ask GitHub Support to remove cached views of the pre-rewrite commit in `mattlau95/ollae`.

---

## iPhone checklist

On the live case study page. Framing only works from the allowlisted origins, so this can't be tested from a phone before then.

**Guestbook frame**

- [ ] Loads when you tap the placeholder.
- [ ] Fits its content: no inner scrollbar, no gap, nothing cut off.
- [ ] "Show all" grows the frame; "Show less" shrinks it.
- [ ] "Matthew L" + I'm in shows the "already on the list" message; your RSVP is unchanged.
- [ ] Tapping the name field doesn't zoom or jump the page.
- [ ] The ollae logo opens a new tab.

**Create frame**

- [ ] Placeholder reads "Board game night @ Alexander Library at 12:30pm".
- [ ] The portfolio's prefill fills the box without submitting.
- [ ] Create Event fills the details, with a today/tomorrow date hint.
- [ ] The share screen shows the link-preview card and caption.
- [ ] "tap to copy" works, and both "Open ↗" links open new tabs.
- [ ] The frame grows without the page jumping.

Most likely to differ in Safari: clipboard access inside a frame, and height updates. Anything created in the create frame is a demo, deleted after 3 days.

---

*Stack: Go 1.26.3 · React 19 · TypeScript · Vite · Tailwind CSS v4 · Fly.io · Fly Postgres · Vercel · Resend · Claude Haiku 4.5*
*Tools: Claude Code · Linear · Playwright*
