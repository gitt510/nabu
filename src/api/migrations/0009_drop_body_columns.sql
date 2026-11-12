-- A task is its title, its metadata, and its memo: the body sections go, with the lines old tasks wrote in
-- them. Every write before this one keeps those lines in events.
ALTER TABLE tasks DROP COLUMN situation;
ALTER TABLE tasks DROP COLUMN complication;
ALTER TABLE tasks DROP COLUMN question;
ALTER TABLE tasks DROP COLUMN answer;
ALTER TABLE tasks DROP COLUMN why;
ALTER TABLE tasks DROP COLUMN dod;
