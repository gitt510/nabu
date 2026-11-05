-- A task carries a memo: lines its owner adds while working on it, such as what was done or thought.
ALTER TABLE tasks ADD COLUMN memo TEXT NOT NULL DEFAULT '[]';
