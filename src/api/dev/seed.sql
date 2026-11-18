-- What just dev starts from: made-up rows, at least five in every table, on an empty local D1, so the viewer
-- shows every state without a copy of the production tasks. Schedules and habit checks count from today in
-- Asia/Tokyo, so the day the seed runs always has something late, due, and ahead.
INSERT INTO tasks (id, title, status, area, project, scheduled, waiting, why_canceled, tickets, prs, links, created, updated, memo) VALUES
  (1, 'viewer の一覧を 1 列にする', 'todo', 'work', 'nabu', NULL, NULL, '[]', '[]', '["https://example.com/nabu/pull/1"]', '[]', '2026-11-18T09:00:00+09:00', '2026-11-18T09:00:00+09:00', '["行をその場で開く形にした", "狭い画面はまだ見ていない"]'),
  (2, 'CI の cache を見直す', 'todo', 'work', 'nabu', strftime('%Y-%m-%dT10:00:00+09:00', 'now', '+9 hours'), NULL, '[]', '[]', '[]', '[]', '2026-11-17T09:00:00+09:00', '2026-11-17T09:00:00+09:00', '[]'),
  (3, 'レビューの返事を待つ', 'todo', 'work', NULL, NULL, 'reviewer の返事', '[]', '["https://example.com/tickets/42"]', '[]', '[]', '2026-11-16T09:00:00+09:00', '2026-11-16T09:00:00+09:00', '[]'),
  (4, '確定申告の書類をそろえる', 'todo', 'personal', NULL, date('now', '+9 hours', '-3 days'), NULL, '[]', '[]', '[]', '["https://example.com/tax"]', '2026-11-15T09:00:00+09:00', '2026-11-15T09:00:00+09:00', '[]'),
  (5, '本棚を片づける', 'todo', 'personal', NULL, date('now', '+9 hours'), NULL, '[]', '[]', '[]', '[]', '2026-11-14T09:00:00+09:00', '2026-11-14T09:00:00+09:00', '[]'),
  (6, '古いブランチを消す', 'todo', 'work', NULL, NULL, NULL, '[]', '[]', '[]', '[]', '2026-11-13T09:00:00+09:00', '2026-11-13T09:00:00+09:00', '[]'),
  (7, '歯医者を予約する', 'todo', 'personal', NULL, strftime('%Y-%m-%dT18:30:00+09:00', 'now', '+9 hours', '+1 day'), NULL, '[]', '[]', '[]', '[]', '2026-11-12T09:00:00+09:00', '2026-11-12T09:00:00+09:00', '[]'),
  (8, 'API の schema を書き出す', 'done', 'work', 'nabu', date('now', '+9 hours', '-1 day'), NULL, '[]', '[]', '["https://example.com/nabu/pull/2"]', '[]', '2026-11-11T09:00:00+09:00', '2026-11-12T18:00:00+09:00', '["openapi.json を生成した"]'),
  (9, 'login を cloudflared 越しにする', 'done', 'work', 'nabu', NULL, NULL, '[]', '[]', '[]', '[]', '2026-11-09T09:00:00+09:00', '2026-11-10T12:00:00+09:00', '[]'),
  (10, '別の task 管理 tool を試す', 'canceled', 'personal', NULL, NULL, NULL, '["今の tool で足りている"]', '[]', '[]', '[]', '2026-11-08T09:00:00+09:00', '2026-11-11T09:00:00+09:00', '[]');

-- Every task begins with its create event, the row as the API would have stored it.
INSERT INTO events (task_id, at, action, before, after)
  SELECT id, created, 'create', NULL,
         json_object('title', title, 'status', status, 'area', area, 'project', project, 'scheduled', scheduled,
                     'waiting', waiting, 'tickets', json(tickets), 'prs', json(prs), 'links', json(links),
                     'why_canceled', json(why_canceled), 'memo', json(memo), 'created', created,
                     'updated', updated)
  FROM tasks ORDER BY id;

INSERT INTO items (task_id, position, text, done, created, updated) VALUES
  (1, 1, 'mockup を比べる', 1, '2026-11-18T09:00:00+09:00', '2026-11-18T09:00:00+09:00'),
  (1, 2, '行を開く部品を作る', 1, '2026-11-18T09:00:00+09:00', '2026-11-18T09:00:00+09:00'),
  (1, 3, '狭い画面で確かめる', 0, '2026-11-18T09:00:00+09:00', '2026-11-18T09:00:00+09:00'),
  (2, 1, 'pnpm の store を cache する', 0, '2026-11-17T09:00:00+09:00', '2026-11-17T09:00:00+09:00'),
  (2, 2, 'Go の module を cache する', 0, '2026-11-17T09:00:00+09:00', '2026-11-17T09:00:00+09:00'),
  (4, 1, '源泉徴収票', 1, '2026-11-15T09:00:00+09:00', '2026-11-15T09:00:00+09:00'),
  (4, 2, '医療費の領収書', 0, '2026-11-15T09:00:00+09:00', '2026-11-15T09:00:00+09:00'),
  (8, 1, 'route を zod で書く', 1, '2026-11-11T09:00:00+09:00', '2026-11-12T18:00:00+09:00');

INSERT INTO habits (id, title, created) VALUES
  (1, '散歩する', '2026-11-01T09:00:00+09:00'),
  (2, '本を 10 ページ読む', '2026-11-01T09:00:00+09:00'),
  (3, '日記を書く', '2026-11-01T09:00:00+09:00'),
  (4, 'ストレッチする', '2026-11-05T09:00:00+09:00'),
  (5, '水を 2 リットル飲む', '2026-11-10T09:00:00+09:00');

INSERT INTO habit_checks (habit_id, date) VALUES
  (1, date('now', '+9 hours')),
  (3, date('now', '+9 hours')),
  (1, date('now', '+9 hours', '-1 day')),
  (2, date('now', '+9 hours', '-1 day')),
  (4, date('now', '+9 hours', '-1 day')),
  (1, date('now', '+9 hours', '-2 days')),
  (5, date('now', '+9 hours', '-2 days'));
