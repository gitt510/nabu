// Builds dist/index.html from data.json, the output of `nabu task ls --json`.
import { execFileSync } from "node:child_process";
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { homedir } from "node:os";
import { marked } from "marked";

const here = (name) => new URL(name, import.meta.url);
const read = (name) => readFileSync(here(name), "utf8");
// The mark: the mušḫuššu, the dragon Nabu stands on, drawn in currentColor. The header colors it from the page;
// the favicon carries its own colors, switching with the browser's color scheme.
const icon = read("icon.svg").trim();
const faviconStyle = "<style>svg{color:#1b1c1f}@media (prefers-color-scheme:dark){svg{color:#ececef}}</style>";
const favicon = `data:image/svg+xml,${encodeURIComponent(icon.replace(/^(<svg[^>]*>)/, `$1${faviconStyle}`))}`;

// The notes repository, from nabu's own config; absent (a build away from that machine) leaves paths relative.
const root = (() => {
  try {
    const r = JSON.parse(readFileSync(`${homedir()}/.config/nabu/config.json`, "utf8")).root ?? "";
    return r.replace(/^~(?=\/|$)/, homedir()).replace(/\/$/, "");
  } catch { return ""; }
})();

// When each task was last committed, from the notes repository's history (newest first, so the first
// mention of a path wins). Empty away from that machine; the viewer then sorts those tasks last.
const updated = (() => {
  if (!root) return {};
  try {
    const log = execFileSync("git", ["-C", root, "log", "--format=%x00%cI", "--name-only", "--", "tasks/"], { encoding: "utf8" });
    const at = {};
    let when = "";
    for (const line of log.split("\n")) {
      if (line.startsWith("\0")) when = line.slice(1);
      else if (line && !(line in at)) at[line] = when;
    }
    return at;
  } catch { return {}; }
})();

const tasks = JSON.parse(read("data.json")).map((r) => {
  const frontmatter = r.frontmatter ?? {};
  let body = r.body.trim();
  if (body.startsWith("# " + r.title)) body = body.slice(r.title.length + 2);
  body = body.trim();
  return {
    path: r.path,
    file: root ? `${root}/${r.path}` : r.path, // what the crumb copies
    slug: r.slug,
    status: r.status,
    title: r.title,
    frontmatter,
    updated: updated[r.path] ?? "",
    html: marked.parse(body),
    text: [r.title, r.slug, frontmatter.waiting ?? "", frontmatter.project ?? "", body].join("\n").toLowerCase(),
  };
});
tasks.sort((a, b) => (a.slug < b.slug ? -1 : a.slug > b.slug ? 1 : 0)); // the viewer orders; this only fixes ties

// escaping < keeps the JSON from closing its <script>
const data = JSON.stringify(tasks).replaceAll("<", "\\u003c");
const css = read("app.css");
const js = read("app.js");

const page = `<!doctype html>
<html lang="ja">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>nabu</title>
<link rel="icon" href="${favicon}">
<script>{ let t; try { t = JSON.parse(localStorage.getItem("nabu-view") ?? "{}").theme; } catch {} document.documentElement.classList.toggle("dark", t ? t === "dark" : matchMedia("(prefers-color-scheme: dark)").matches); }</script>
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
    <button type="button" id="theme-btn"></button>
    <button type="button" id="help-btn" title="keyboard shortcuts (?)" aria-label="ショートカット一覧"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="10"/><path d="M9.1 9a3 3 0 0 1 5.8 1c0 2-3 3-3 3"/><path d="M12 17h.01"/></svg></button>
    <button type="button" id="settings-btn" title="settings (,)" aria-label="設定"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12.2 2h-.4a2 2 0 0 0-2 2v.2a2 2 0 0 1-1 1.7l-.4.3a2 2 0 0 1-2 0l-.2-.1a2 2 0 0 0-2.7.7l-.2.4a2 2 0 0 0 .7 2.7l.2.1a2 2 0 0 1 1 1.7v.6a2 2 0 0 1-1 1.7l-.2.1a2 2 0 0 0-.7 2.7l.2.4a2 2 0 0 0 2.7.7l.2-.1a2 2 0 0 1 2 0l.4.3a2 2 0 0 1 1 1.7V20a2 2 0 0 0 2 2h.4a2 2 0 0 0 2-2v-.2a2 2 0 0 1 1-1.7l.4-.3a2 2 0 0 1 2 0l.2.1a2 2 0 0 0 2.7-.7l.2-.4a2 2 0 0 0-.7-2.7l-.2-.1a2 2 0 0 1-1-1.7v-.6a2 2 0 0 1 1-1.7l.2-.1a2 2 0 0 0 .7-2.7l-.2-.4a2 2 0 0 0-2.7-.7l-.2.1a2 2 0 0 1-2 0l-.4-.3a2 2 0 0 1-1-1.7V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg></button>
  </header>
  <aside class="side" id="side">
    <nav class="views" id="views" aria-label="一覧の切り替え"></nav>
  </aside>
  <section class="content">
    <div class="list" id="list">
      <label class="search"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/></svg><input id="q" type="search" placeholder="Search" autocomplete="off" aria-label="Search"></label>
      <div class="list-head"><span class="n" id="count"></span><div class="sort" id="sort" role="group" aria-label="並び順"><button type="button" tabindex="-1" data-sort="updated" title="updated (s)">updated</button><button type="button" tabindex="-1" data-sort="created" title="created (s)">created</button></div></div>
      <ul class="files" id="files"></ul>
    </div>
    <main class="main"><article class="doc" id="doc"><p class="empty">読み込み中</p></article></main>
  </section>
</div>
<dialog id="help"><button class="close" type="button" aria-label="閉じる">Esc</button><h2>Keyboard</h2><dl class="keys" id="keys"></dl></dialog>
<dialog id="settings"><button class="close" type="button" aria-label="閉じる">Esc</button><h2>Settings</h2>
  <div class="settings">
    <span class="label" id="s-theme-label">テーマ</span><div class="theme" id="s-theme" role="group" aria-labelledby="s-theme-label"><button type="button" data-theme="">OS に合わせる</button><button type="button" data-theme="light">light</button><button type="button" data-theme="dark">dark</button></div>
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
