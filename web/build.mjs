// Builds dist/index.html and dist/marked.js, the viewer without its data; the page fetches every task
// from /api/tasks and renders the markdown itself.
import { copyFileSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { createRequire } from "node:module";
import { AREAS, LINE_LIMIT, LISTS, SECTIONS, STATUSES } from "./src/schema.ts";

const here = (name) => new URL(name, import.meta.url);
const read = (name) => readFileSync(here(name), "utf8");
// The mark: the mušḫuššu, the dragon Nabu stands on, drawn in currentColor. The header colors it from the page;
// the favicon carries its own colors, switching with the browser's color scheme.
const icon = read("icon.svg").trim();
const faviconStyle = "<style>svg{color:#1b1c1f}@media (prefers-color-scheme:dark){svg{color:#ececef}}</style>";
const favicon = `data:image/svg+xml,${encodeURIComponent(icon.replace(/^(<svg[^>]*>)/, `$1${faviconStyle}`))}`;

const css = read("app.css");
// The page takes the task contract from schema.ts: these lines go above app.js, which uses them as its own.
const heading = (k) => k[0].toUpperCase() + k.slice(1).replace("_", " ");
const contract = [
  `const AREAS = ${JSON.stringify(AREAS)};`,
  `const STATUSES = ${JSON.stringify(STATUSES)};`,
  `const LISTS = ${JSON.stringify(LISTS)};`,
  `const SECTIONS = ${JSON.stringify(SECTIONS.map((k) => [heading(k), k]))};`,
].join("\n");
const js = `${contract}\n${read("app.js")}`;

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
<dialog id="drop"><button class="close" type="button" aria-label="閉じる">Esc</button><h2>Drop</h2>
  <form class="drop" id="drop-form">
    <label for="drop-why">やらない理由</label>
    <p class="hint" id="drop-hint">1 行 1 項目、${LINE_LIMIT} 字まで</p>
    <textarea id="drop-why" rows="3" aria-describedby="drop-hint"></textarea>
    <p class="error" id="drop-error" role="alert"></p>
    <p class="settings-foot"><button type="submit">dropped にする</button></p>
  </form>
</dialog>
<script type="module">
${js}
</script>
</body>
</html>
`;

mkdirSync(here("dist/"), { recursive: true });
writeFileSync(here("dist/index.html"), page);
copyFileSync(createRequire(import.meta.url).resolve("marked").replace(/marked\.cjs$/, "marked.esm.js"), here("dist/marked.js"));
