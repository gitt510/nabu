-- The body becomes Why and DoD: why the task is worth starting, and what closes it. The SCQA columns stay
-- with the lines old tasks wrote, though the API no longer reads or writes them: a schema change adds
-- columns and leaves old rows as they were.
ALTER TABLE tasks ADD COLUMN why TEXT NOT NULL DEFAULT '[]';
ALTER TABLE tasks ADD COLUMN dod TEXT NOT NULL DEFAULT '[]';
