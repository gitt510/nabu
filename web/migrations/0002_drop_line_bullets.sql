-- A section line is one item already; drop the "- " each line carried from the markdown body. Events keep
-- the rows as they were written.
UPDATE tasks SET
  situation    = (SELECT json_group_array(CASE WHEN value LIKE '- %' THEN substr(value, 3) ELSE value END) FROM (SELECT value FROM json_each(tasks.situation) ORDER BY key)),
  complication = (SELECT json_group_array(CASE WHEN value LIKE '- %' THEN substr(value, 3) ELSE value END) FROM (SELECT value FROM json_each(tasks.complication) ORDER BY key)),
  question     = (SELECT json_group_array(CASE WHEN value LIKE '- %' THEN substr(value, 3) ELSE value END) FROM (SELECT value FROM json_each(tasks.question) ORDER BY key)),
  answer       = (SELECT json_group_array(CASE WHEN value LIKE '- %' THEN substr(value, 3) ELSE value END) FROM (SELECT value FROM json_each(tasks.answer) ORDER BY key)),
  why_dropped  = (SELECT json_group_array(CASE WHEN value LIKE '- %' THEN substr(value, 3) ELSE value END) FROM (SELECT value FROM json_each(tasks.why_dropped) ORDER BY key));
