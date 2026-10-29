-- A task's id is the number people copy and paste, so it must never pass to another task. AUTOINCREMENT
-- stops SQLite from handing out the number of the newest deleted task again; SQLite cannot add it to a
-- table, so the table is built anew with every row keeping its id.
CREATE TABLE tasks_new (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
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

INSERT INTO tasks_new SELECT * FROM tasks;
DROP TABLE tasks;
ALTER TABLE tasks_new RENAME TO tasks;

-- A task deleted before this migration left its number in events only. A row that takes the highest such
-- number and goes again moves AUTOINCREMENT past it, so that number is never handed out either.
INSERT INTO tasks (id, slug, title, status, area, created, updated)
  SELECT max(task_id), 'migration-0004', '-', 'inbox', 'work', '', '' FROM events
  HAVING max(task_id) > coalesce((SELECT max(id) FROM tasks), 0);
DELETE FROM tasks WHERE slug = 'migration-0004';
