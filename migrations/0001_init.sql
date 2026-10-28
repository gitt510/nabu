-- A task is one row; its status is a column, and each SCQA section is a JSON array of markdown lines.
CREATE TABLE tasks (
  id           INTEGER PRIMARY KEY,
  slug         TEXT NOT NULL UNIQUE,
  title        TEXT NOT NULL,
  status       TEXT NOT NULL CHECK (status IN ('inbox', 'doing', 'done', 'dropped')),
  area         TEXT NOT NULL CHECK (area IN ('work', 'personal')),
  project      TEXT,
  scheduled    TEXT,
  waiting      TEXT,
  situation    TEXT NOT NULL DEFAULT '[]',
  complication TEXT NOT NULL DEFAULT '[]',
  question     TEXT NOT NULL DEFAULT '[]',
  answer       TEXT NOT NULL DEFAULT '[]',
  why_dropped  TEXT NOT NULL DEFAULT '[]',
  tickets      TEXT NOT NULL DEFAULT '[]',
  prs          TEXT NOT NULL DEFAULT '[]',
  links        TEXT NOT NULL DEFAULT '[]',
  created      TEXT NOT NULL,
  updated      TEXT NOT NULL,
  CHECK (status <> 'dropped' OR why_dropped <> '[]')
);

-- Every write appends one event with the row before and after it, so history outlives D1 Time Travel.
-- task_id carries no foreign key: a deleted task keeps its events.
CREATE TABLE events (
  id      INTEGER PRIMARY KEY,
  task_id INTEGER NOT NULL,
  at      TEXT NOT NULL,
  action  TEXT NOT NULL CHECK (action IN ('create', 'update', 'delete')),
  before  TEXT,
  after   TEXT
);

CREATE INDEX events_task_id ON events (task_id);
