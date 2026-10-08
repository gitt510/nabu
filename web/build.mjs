// Builds dist/index.html from data.json, the output of `nabu task ls --json`.
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { marked } from "marked";

const here = (name) => new URL(name, import.meta.url);
const read = (name) => readFileSync(here(name), "utf8");
// The mark: the mušḫuššu, the dragon Nabu stands on, drawn in currentColor. The header colors it from the page;
// the favicon carries its own colors, switching with the browser's color scheme.
const icon = read("icon.svg").trim();
const faviconStyle = "<style>svg{color:#1b1c1f}@media (prefers-color-scheme:dark){svg{color:#ececef}}</style>";
const favicon = `data:image/svg+xml,${encodeURIComponent(icon.replace(/^(<svg[^>]*>)/, `$1${faviconStyle}`))}`;

const order = { doing: 0, inbox: 1, done: 2, stray: 3 };
// scheduled tasks first, by instant (RFC3339 with any offset; a date alone is the start of that day in Asia/Tokyo), then the rest by slug
const instant = (s) => Date.parse(/^\d{4}-\d{2}-\d{2}$/.test(s) ? `${s}T00:00:00+09:00` : s);
const key = (t) => (t.frontmatter.scheduled ? "0" + String(instant(t.frontmatter.scheduled)).padStart(15, "0") : "1" + t.slug);

const tasks = JSON.parse(read("data.json")).map((r) => {
  const frontmatter = r.frontmatter ?? {};
  let body = r.body.trim();
  if (body.startsWith("# " + r.title)) body = body.slice(r.title.length + 2);
  body = body.trim();
  return {
    path: r.path,
    slug: r.slug,
    status: r.status,
    title: r.title,
    frontmatter,
    html: marked.parse(body),
    text: [r.title, r.slug, frontmatter.waiting ?? "", frontmatter.project ?? "", body].join("\n").toLowerCase(),
  };
});
// doing, inbox, done, stray; inside a status, scheduled tasks by time, then the rest by slug
tasks.sort((a, b) => order[a.status] - order[b.status] || (key(a) < key(b) ? -1 : key(a) > key(b) ? 1 : 0));

// escaping < keeps the JSON from closing its <script>
const data = JSON.stringify(tasks).replaceAll("<", "\\u003c");
const css = read("app.css");
const js = read("app.js");
const built = new Intl.DateTimeFormat("ja-JP", {
  timeZone: "Asia/Tokyo",
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit",
}).format(new Date());

const page = `<!doctype html>
<html lang="ja">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>nabu</title>
<link rel="icon" href="${favicon}">
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Noto+Sans+JP:wght@400;500;700&family=Noto+Sans+Mono:wght@400&display=swap" rel="stylesheet">
<style>
${css}
</style>
</head>
<body>
<div class="shell">
  <header class="top">
    <button type="button" id="side-btn" class="menu" title="sidebar" aria-label="一覧" aria-controls="side" aria-expanded="false"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><path d="M4 7h16M4 12h16M4 17h16"/></svg></button>
    <button type="button" id="home-btn" class="brand" title="Home (H)">${icon}<span>nabu</span></button>
    <span class="spacer"></span>
    <button type="button" id="help-btn" title="keyboard shortcuts (?)" aria-label="ショートカット一覧"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="10"/><path d="M9.1 9a3 3 0 0 1 5.8 1c0 2-3 3-3 3"/><path d="M12 17h.01"/></svg></button>
    <button type="button" id="settings-btn" title="settings (,)" aria-label="設定"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12.2 2h-.4a2 2 0 0 0-2 2v.2a2 2 0 0 1-1 1.7l-.4.3a2 2 0 0 1-2 0l-.2-.1a2 2 0 0 0-2.7.7l-.2.4a2 2 0 0 0 .7 2.7l.2.1a2 2 0 0 1 1 1.7v.6a2 2 0 0 1-1 1.7l-.2.1a2 2 0 0 0-.7 2.7l.2.4a2 2 0 0 0 2.7.7l.2-.1a2 2 0 0 1 2 0l.4.3a2 2 0 0 1 1 1.7V20a2 2 0 0 0 2 2h.4a2 2 0 0 0 2-2v-.2a2 2 0 0 1 1-1.7l.4-.3a2 2 0 0 1 2 0l.2.1a2 2 0 0 0 2.7-.7l.2-.4a2 2 0 0 0-.7-2.7l-.2-.1a2 2 0 0 1-1-1.7v-.6a2 2 0 0 1 1-1.7l.2-.1a2 2 0 0 0 .7-2.7l-.2-.4a2 2 0 0 0-2.7-.7l-.2.1a2 2 0 0 1-2 0l-.4-.3a2 2 0 0 1-1-1.7V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg></button>
  </header>
  <aside class="side" id="side">
    <label class="search"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/></svg><input id="q" type="search" placeholder="Search" autocomplete="off" aria-label="Search"></label>
    <div class="facets" id="facets"></div>
    <ul class="tree" id="tree"></ul>
    <div class="side-foot">built ${built}</div>
  </aside>
  <main class="main"><article class="doc" id="doc"><p class="empty">読み込み中</p></article></main>
</div>
<dialog id="help"><button class="close" type="button" aria-label="閉じる">Esc</button><h2>Keyboard</h2><dl class="keys" id="keys"></dl></dialog>
<dialog id="settings"><button class="close" type="button" aria-label="閉じる">Esc</button><h2>Settings</h2>
  <div class="settings">
    <label for="s-font">文字サイズ</label><input id="s-font" type="range" min="14" max="24" step="1" data-key="font"><output for="s-font"></output>
    <label for="s-side">sidebar 幅</label><input id="s-side" type="range" min="220" max="480" step="20" data-key="side"><output for="s-side"></output>
    <label for="s-width">本文幅</label><input id="s-width" type="range" min="36" max="72" step="2" data-key="width"><output for="s-width"></output>
  </div>
  <p class="settings-foot"><button type="button" id="s-reset">default に戻す</button></p>
</dialog>
<script id="data" type="application/json">${data}</script>
<script>
${js}
</script>
</body>
</html>
`;

mkdirSync(here("dist/"), { recursive: true });
writeFileSync(here("dist/index.html"), page);
