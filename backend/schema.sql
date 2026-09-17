-- Matches production (database showup_backend on showup-db), dumped
-- 2026-09-17 with pg_dump --schema-only.

CREATE TABLE IF NOT EXISTS events (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug        TEXT NOT NULL UNIQUE,
  title       TEXT NOT NULL,
  location    TEXT,
  event_date  TIMESTAMPTZ,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  emoji       TEXT NOT NULL DEFAULT '',
  admin_token TEXT NOT NULL DEFAULT '',
  -- Created from the portfolio embed; deleted 3 days after creation.
  is_demo     BOOLEAN NOT NULL DEFAULT false,
  -- A name already on the list can't be resubmitted to change its answer.
  append_only BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE IF NOT EXISTS responses (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id    UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  name        TEXT NOT NULL,
  status      TEXT NOT NULL CHECK (status IN ('in', 'out', 'remind_me')),
  notify_via  TEXT,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  guests      INT NOT NULL DEFAULT 0,
  reminded_at TIMESTAMPTZ
);

-- One response per name per event, case-insensitive. An expression can't be
-- a table constraint, so this is a unique index.
CREATE UNIQUE INDEX IF NOT EXISTS responses_event_id_name_idx
  ON responses (event_id, lower(name));

CREATE TABLE IF NOT EXISTS settings (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

-- Claude API calls per UTC day, checked against CLAUDE_DAILY_CAP.
CREATE TABLE IF NOT EXISTS claude_usage (
  day   DATE PRIMARY KEY,
  calls INT NOT NULL DEFAULT 0
);
