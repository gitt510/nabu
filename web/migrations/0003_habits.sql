-- A habit is something done every day.
CREATE TABLE habits (
  id      INTEGER PRIMARY KEY,
  title   TEXT NOT NULL,
  created TEXT NOT NULL
);

-- A check is one day a habit was done, as a date in Asia/Tokyo: a row means done, no row means not yet.
CREATE TABLE habit_checks (
  habit_id INTEGER NOT NULL REFERENCES habits (id),
  date     TEXT NOT NULL,
  PRIMARY KEY (habit_id, date)
);
