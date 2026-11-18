-- A task is named by its id alone: the slug goes. SQLite cannot drop a UNIQUE column, so the table is built
-- anew with every row keeping its id. items points at tasks, and dropping tasks under it fails the foreign
-- key even deferred, so items steps aside into a copy and comes back as 0011 made it.

CREATE TABLE tasks_new (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  title        TEXT NOT NULL,
  status       TEXT NOT NULL CHECK (status IN ('todo', 'done', 'canceled')),
  area         TEXT NOT NULL CHECK (area IN ('work', 'personal')),
  project      TEXT,
  scheduled    TEXT,
  waiting      TEXT,
  why_canceled TEXT NOT NULL DEFAULT '[]',
  tickets      TEXT NOT NULL DEFAULT '[]',
  prs          TEXT NOT NULL DEFAULT '[]',
  links        TEXT NOT NULL DEFAULT '[]',
  created      TEXT NOT NULL,
  updated      TEXT NOT NULL,
  memo         TEXT NOT NULL DEFAULT '[]',
  CHECK ((status = 'canceled') = (why_canceled <> '[]'))
);

INSERT INTO tasks_new
  SELECT id, title, status, area, project, scheduled, waiting, why_canceled, tickets, prs, links, created,
         updated, memo
  FROM tasks;
CREATE TABLE items_copy AS SELECT * FROM items;
DROP TABLE items;
DROP TABLE tasks;
ALTER TABLE tasks_new RENAME TO tasks;

CREATE TABLE items (
  id       INTEGER PRIMARY KEY,
  task_id  INTEGER NOT NULL REFERENCES tasks (id),
  position REAL NOT NULL,
  text     TEXT NOT NULL,
  done     INTEGER NOT NULL DEFAULT 0 CHECK (done IN (0, 1)),
  created  TEXT NOT NULL,
  updated  TEXT NOT NULL
);
CREATE INDEX items_task ON items (task_id, position);
INSERT INTO items SELECT * FROM items_copy;
DROP TABLE items_copy;

-- The new table counts ids from the rows it took, so a deleted task's number would come back. A row that
-- takes the highest number any task ever held and goes again moves AUTOINCREMENT past it, as 0011 did.
INSERT INTO tasks (id, title, status, area, created, updated)
  SELECT max(task_id), 'migration-0012', 'todo', 'work', '', '' FROM events
  HAVING max(task_id) > coalesce((SELECT max(id) FROM tasks), 0);
DELETE FROM tasks WHERE title = 'migration-0012' AND created = '';
