CREATE TABLE IF NOT EXISTS events (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug        TEXT NOT NULL UNIQUE,
  title       TEXT NOT NULL,
  location    TEXT,
  event_date  TIMESTAMPTZ,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  admin_token TEXT NOT NULL DEFAULT '',
  emoji       TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS responses (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id   UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  name       TEXT NOT NULL,
  status     TEXT NOT NULL CHECK (status IN ('in', 'out', 'remind_me')),
  notify_via TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (event_id, lower(name))
);
