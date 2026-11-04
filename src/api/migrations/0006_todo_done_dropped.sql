-- A task is todo, done, or dropped, like a checkbox that can also be crossed out: inbox and doing become
-- todo, since scheduled says when the work happens. Only a dropped task says why_dropped. SQLite cannot
-- change a CHECK, so the table is built anew with every row keeping its id; the SCQA columns stay with the
-- lines old tasks wrote.
CREATE TABLE tasks_new (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  slug         TEXT NOT NULL UNIQUE,
  title        TEXT NOT NULL,
  status       TEXT NOT NULL CHECK (status IN ('todo', 'done', 'dropped')),
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
  why          TEXT NOT NULL DEFAULT '[]',
  dod          TEXT NOT NULL DEFAULT '[]',
  CHECK ((status = 'dropped') = (why_dropped <> '[]'))
);

INSERT INTO tasks_new
  SELECT id, slug, title, CASE WHEN status IN ('inbox', 'doing') THEN 'todo' ELSE status END, area, project,
         scheduled, waiting, situation, complication, question, answer, why_dropped, tickets, prs, links, created,
         updated, why, dod
  FROM tasks;
DROP TABLE tasks;
ALTER TABLE tasks_new RENAME TO tasks;

-- The new table counts ids from the rows it took, so a deleted task's number would come back. A row that
-- takes the highest number any task ever held and goes again moves AUTOINCREMENT past it, as 0004 did.
INSERT INTO tasks (id, slug, title, status, area, created, updated)
  SELECT max(task_id), 'migration-0006', '-', 'todo', 'work', '', '' FROM events
  HAVING max(task_id) > coalesce((SELECT max(id) FROM tasks), 0);
DELETE FROM tasks WHERE slug = 'migration-0006';
