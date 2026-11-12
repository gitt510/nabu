-- An item is one line of a task's checklist, in the order position gives. The outline it replaces goes with
-- its lines: a date belongs to the task, and a line under no task has no place.
DROP TABLE items;
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
