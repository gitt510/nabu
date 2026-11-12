-- An item is one line of an outline: a line under no parent is a root, and a line under a parent is one of
-- its children, in the order position gives. due is the day a root is for, so a month or a week is the roots
-- due in it. done is only read on a line with no children; a line with children is done when they all are.
CREATE TABLE items (
  id        INTEGER PRIMARY KEY,
  parent_id INTEGER REFERENCES items (id),
  position  REAL NOT NULL,
  text      TEXT NOT NULL,
  done      INTEGER NOT NULL DEFAULT 0 CHECK (done IN (0, 1)),
  due       TEXT,
  created   TEXT NOT NULL,
  updated   TEXT NOT NULL
);
CREATE INDEX items_parent ON items (parent_id, position);
