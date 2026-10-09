const tasks = JSON.parse(document.getElementById("data").textContent);
const byId = Object.fromEntries(tasks.map((t) => [t.slug, t]));
const STATUS = ["doing", "inbox", "done", "dropped"];
const $ = (id) => document.getElementById(id);
const tree = $("tree");
const doc = $("doc");
const q = $("q");
const main = document.querySelector(".main");
const facets = $("facets");
const pick = { area: "", project: "" }; // "" lets every task through

// scheduled is RFC3339 with an offset, or a date alone (YYYY-MM-DD) for the whole day in Asia/Tokyo
const dateOnly = (s) => /^\d{4}-\d{2}-\d{2}$/.test(s);
const instant = (s) => new Date(dateOnly(s) ? `${s}T00:00:00+09:00` : s);
const fmt = (iso) => {
  const d = instant(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const day = { timeZone: "Asia/Tokyo", month: "numeric", day: "numeric" };
  return dateOnly(iso) ? d.toLocaleDateString("ja-JP", day) : d.toLocaleString("ja-JP", { ...day, hour: "2-digit", minute: "2-digit" });
};
const ticket = (u) => {
  try {
    const url = new URL(u);
    if (url.hostname.includes("wrike")) return `Wrike ${url.searchParams.get("id") ?? ""}`.trim();
    if (url.hostname === "github.com") { const [, , repo, , num] = url.pathname.split("/"); return num ? `${repo}#${num}` : repo; }
    return url.hostname;
  } catch { return u; }
};
const el = (tag, attrs = {}, ...kids) => {
  const e = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs)) k === "class" ? (e.className = v) : k.startsWith("on") ? e.addEventListener(k.slice(2), v) : e.setAttribute(k, v);
  e.append(...kids);
  return e;
};

// ---- sections -------------------------------------------------------------
// Frontmatter sections first, then one section per status folder. A task may
// sit in both (a doing task with a date is in Scheduled and in Doing).
const open_ = (t) => t.status !== "done" && t.status !== "dropped";
const bySchedule = (a, b) => instant(a.frontmatter.scheduled) - instant(b.frontmatter.scheduled);
const SECTIONS = [
  { id: "scheduled", label: "Scheduled", pick: (t) => open_(t) && t.frontmatter.scheduled, sort: bySchedule },
  { id: "waiting", label: "Waiting", pick: (t) => open_(t) && t.frontmatter.waiting },
  { id: "doing", label: "Doing", pick: (t) => t.status === "doing" },
  { id: "inbox", label: "Inbox", pick: (t) => t.status === "inbox" },
  { id: "done", label: "Done", pick: (t) => t.status === "done", closed: true },
  { id: "dropped", label: "Dropped", pick: (t) => t.status === "dropped", closed: true },
];
const PROPS = ["scheduled", "waiting", "area", "project", "tickets", "prs", "links", "created"];
const today = new Date().toLocaleDateString("ja-JP", { timeZone: "Asia/Tokyo" });
const dayOf = (iso) => instant(iso).toLocaleDateString("ja-JP", { timeZone: "Asia/Tokyo" });
const isLate = (iso) => instant(iso) < new Date() && dayOf(iso) !== today;

function fileButton(t) {
  const fm = t.frontmatter;
  const b = el("button", { class: "file", "data-slug": t.slug, type: "button", tabindex: "-1", title: t.slug, onclick: () => select(t.slug) }, t.title);
  if (fm.scheduled) b.append(el("span", { class: `flag${isLate(fm.scheduled) ? " late" : ""}` }, fmt(fm.scheduled).split(" ")[0]));
  else if (fm.waiting) b.append(el("span", { class: "flag" }, "⏳"));
  return b;
}

function renderTree() {
  tree.replaceChildren();
  for (const sec of SECTIONS) {
    const files = tasks.filter(sec.pick);
    if (sec.sort) files.sort(sec.sort);
    const dir = el("li", { class: `dir ${sec.id}${sec.closed ? " closed" : ""}` });
    const name = el("div", { class: "dir-name", onclick: () => dir.classList.toggle("closed") }, sec.label, el("span", { class: "n" }, String(files.length)));
    const ul = el("ul", { class: "files" });
    for (const t of files) ul.append(el("li", {}, fileButton(t)));
    dir.append(name, ul);
    tree.append(dir);
  }
}

// ---- home -------------------------------------------------------------------
function renderHome() {
  const link = (t) => el("li", {}, el("a", { href: `#${t.slug}`, onclick: (e) => { e.preventDefault(); select(t.slug); } }, t.title), t.frontmatter.scheduled ? el("span", { class: `flag${isLate(t.frontmatter.scheduled) ? " late" : ""}` }, fmt(t.frontmatter.scheduled)) : "");
  const list = (items, empty) => (items.length ? el("ul", { class: "home-list" }, ...items.map(link)) : el("p", { class: "empty" }, empty));
  // Today: open tasks scheduled for today, plus the late ones still open.
  const due = tasks.filter((t) => open_(t) && t.frontmatter.scheduled && (isLate(t.frontmatter.scheduled) || dayOf(t.frontmatter.scheduled) === today)).sort(bySchedule);
  // Linked: open tasks with PRs or tickets, PRs first, each link one click away.
  const refs = (t) => [...(t.frontmatter.prs ?? []), ...(t.frontmatter.tickets ?? [])];
  const linked = tasks.filter((t) => open_(t) && refs(t).length).sort((a, b) => !a.frontmatter.prs - !b.frontmatter.prs);
  const row = (t) => el("li", {}, el("a", { href: `#${t.slug}`, onclick: (e) => { e.preventDefault(); select(t.slug); } }, t.title), el("span", { class: "refs" }, ...refs(t).map((u) => el("a", { href: u, target: "_blank", rel: "noopener" }, ticket(u)))));
  doc.replaceChildren(
    el("p", { class: "crumb" }, today), el("h1", {}, "Today"), list(due, "今日のタスクはない"),
    el("h2", { class: "home-h" }, "Linked"), linked.length ? el("ul", { class: "home-list" }, ...linked.map(row)) : el("p", { class: "empty" }, "PR や ticket のあるタスクはない"),
  );
}

// Visible file buttons, in order. Closed folders count as hidden.
const visibleFiles = () => [...tree.querySelectorAll(".dir:not(.closed):not(.hidden) li:not(.hidden) > .file")];

function focusFile(delta) {
  const items = visibleFiles();
  if (!items.length) return;
  const at = items.indexOf(document.activeElement);
  let next;
  if (delta === "first") next = 0;
  else if (delta === "last") next = items.length - 1;
  else if (at < 0) next = Math.max(0, items.findIndex((b) => b.classList.contains("active")));
  else next = Math.min(items.length - 1, Math.max(0, at + delta));
  for (const b of items) b.tabIndex = -1;
  items[next].tabIndex = 0;
  items[next].focus();
  items[next].scrollIntoView({ block: "nearest" });
  select(items[next].dataset.slug);
}

function select(slug, push = true) {
  for (const b of tree.querySelectorAll(".file")) b.classList.toggle("active", b.dataset.slug === slug);
  if (slug === "") {
    renderHome();
    if (push) history.replaceState(null, "", location.pathname);
    main.scrollTop = 0;
    if (push) closeSide();
    return;
  }
  const t = byId[slug];
  if (!t) return;
  tree.querySelector(`.file.active`)?.closest(".dir")?.classList.remove("closed");
  // Properties: keys verbatim, in PROPS order (what to do next first), not file order.
  const meta = el("dl", { class: "meta" });
  const fm = t.frontmatter;
  const value = (key, v) => {
    if (Array.isArray(v)) return v.map((u) => el("a", { href: u, target: "_blank", rel: "noopener" }, ticket(u)));
    if (key === "created" || key === "scheduled") return [fmt(v)];
    if (key === "area" || key === "project") return [el("button", { class: "chip", type: "button", title: `${v} で絞り込む`, onclick: () => narrowTo(fm.area, key === "project" ? v : "") }, v)];
    return [v];
  };
  for (const key of PROPS) if (fm[key]?.length) meta.append(el("div", {}, el("dt", {}, key), el("dd", {}, ...value(key, fm[key]))));
  const body = el("div", { class: "body" });
  body.innerHTML = t.html;
  doc.replaceChildren(el("p", { class: "crumb" }, crumb(t)), el("h1", {}, t.title), meta, body);
  if (push) history.replaceState(null, "", `#${slug}`);
  main.scrollTop = 0;
  if (push) closeSide();
}

// The path line above the title; a click copies the file's path (absolute when the build knew the root).
function crumb(t) {
  const b = el("button", { class: "copy", type: "button", title: `${t.file}\nclick で copy` }, t.path);
  b.addEventListener("click", () => navigator.clipboard.writeText(t.file).then(() => {
    b.textContent = "copied";
    setTimeout(() => { b.textContent = t.path; }, 1200);
  }));
  return b;
}

// Open the task adjacent to the current one, regardless of focus.
function step(delta) {
  const items = visibleFiles();
  const at = items.findIndex((b) => b.classList.contains("active"));
  const next = items[Math.min(items.length - 1, Math.max(0, at + delta))];
  if (next) { select(next.dataset.slug); next.scrollIntoView({ block: "nearest" }); if (tree.contains(document.activeElement)) next.focus(); }
}

function filter() {
  const words = q.value.trim().toLowerCase().split(/\s+/).filter(Boolean);
  const keep = (t) => words.every((w) => t.text.includes(w)) && (!pick.area || t.frontmatter.area === pick.area) && (!pick.project || t.frontmatter.project === pick.project);
  for (const b of tree.querySelectorAll(".file")) if (b.dataset.slug) b.parentElement.classList.toggle("hidden", !keep(byId[b.dataset.slug]));
  for (const dir of tree.querySelectorAll(".dir")) {
    const n = dir.querySelectorAll("li:not(.hidden)").length;
    dir.querySelector(".n").textContent = String(n);
    dir.classList.toggle("hidden", n === 0);
    if (words.length || pick.area || pick.project) dir.classList.remove("closed");
  }
}

// ---- facets: area, then the projects in it, under the search box -----------
// Each is a single choice; pressing the chosen one again lets everything through.
const AREAS = ["work", "personal"];
function renderFacets() {
  const chip = (label, on, onclick) => el("button", { class: "chip", type: "button", "aria-pressed": String(on), onclick }, label);
  const projects = [...new Set(tasks.filter((t) => !pick.area || t.frontmatter.area === pick.area).map((t) => t.frontmatter.project).filter(Boolean))].sort();
  facets.replaceChildren(
    el("div", { class: "facet" }, ...AREAS.map((a) => chip(a, pick.area === a, () => narrowTo(pick.area === a ? "" : a, "")))),
    el("div", { class: "facet" }, ...projects.map((p) => chip(p, pick.project === p, () => narrowTo(pick.area, pick.project === p ? "" : p)))),
  );
}
function narrowTo(area, project) {
  pick.area = area;
  pick.project = project;
  renderFacets();
  filter();
}

// ---- shortcuts (single keys, outside text fields; ? lists them) ------------
const inTextField = (e) => ["INPUT", "TEXTAREA", "SELECT"].includes(e.target.tagName) || e.target.isContentEditable;
const shortcuts = [
  { group: "移動" },
  { key: "/", label: "検索にフォーカス", run: () => q.focus() },
  { key: "h", label: "一覧にフォーカス", run: () => focusFile(0) },
  { key: "H", label: "Home を開く", run: () => select("") },
  { key: "n", label: "次のタスクを開く", run: () => step(1) },
  { key: "p", label: "前のタスクを開く", run: () => step(-1) },
  { key: "j", label: "本文をスクロール", run: () => main.scrollBy({ top: 80 }) },
  { key: "k", label: "本文をスクロール", run: () => main.scrollBy({ top: -80 }) },
  { group: "一覧内" },
  { key: "j / ↓", label: "下へ", tree: true },
  { key: "k / ↑", label: "上へ", tree: true },
  { key: "gg / G", label: "先頭 / 末尾", tree: true },
  { key: "l / Enter", label: "開く（移動しただけでも切り替わる）", tree: true },
  { key: "Esc", label: "一覧から抜ける", tree: true },
  { group: "その他" },
  { key: "?", label: "この一覧", run: () => toggle($("help")) },
  { key: ",", label: "設定", run: () => toggle($("settings")) },
  { key: "t", label: "light / dark を切り替える", run: () => flipTheme() },
  { key: "Esc", label: "検索と area / project の絞り込みを消す / dialog を閉じる", tree: true },
];
const toggle = (d) => (d.open ? d.close() : d.showModal());
let pendingG = false;

document.addEventListener("keydown", (e) => {
  if (e.isComposing || e.metaKey || e.ctrlKey || e.altKey) return;
  if (document.querySelector("dialog[open]")) return; // dialog handles Esc itself
  if (inTextField(e)) {
    if (e.key === "Escape") { if (q.value || pick.area || pick.project) { q.value = ""; narrowTo("", ""); } else focusFile(0); }
    if (e.key === "Enter" && e.target === q) { e.preventDefault(); focusFile(0); }
    return;
  }
  if (tree.contains(e.target)) {
    const k = e.key;
    if (k === "g") { if (pendingG) { focusFile("first"); pendingG = false; } else pendingG = true; e.preventDefault(); return; }
    pendingG = false;
    if (k === "j" || k === "ArrowDown") focusFile(1);
    else if (k === "k" || k === "ArrowUp") focusFile(-1);
    else if (k === "G") focusFile("last");
    else if (k === "l" || k === "ArrowRight" || k === "Enter") { select(e.target.dataset.slug); }
    else if (k === "Escape") { e.target.blur(); main.focus(); }
    else { pendingG = false; return runGlobal(e); }
    e.preventDefault();
    return;
  }
  runGlobal(e);
});
function runGlobal(e) {
  const s = shortcuts.find((s) => s.key === e.key && s.run);
  if (s) { e.preventDefault(); s.run(); }
}

// ---- help & settings ---------------------------------------------------------
$("keys").append(...shortcuts.flatMap((s) => (s.group ? [el("div", { class: "group" }, s.group)] : [el("dt", {}, s.key), el("dd", {}, s.label)])));
for (const d of document.querySelectorAll("dialog")) d.querySelector(".close").addEventListener("click", () => d.close());
$("home-btn").addEventListener("click", () => select(""));
$("help-btn").addEventListener("click", () => toggle($("help")));
$("settings-btn").addEventListener("click", () => toggle($("settings")));

const SETTINGS = {
  font: { def: 18, apply: (v) => (document.documentElement.style.fontSize = `${v}px`), unit: "px" },
  side: { def: 300, apply: (v) => document.documentElement.style.setProperty("--side-w", `${v}px`), unit: "px" },
  width: { def: 50, apply: (v) => document.documentElement.style.setProperty("--doc-w", `${v}rem`), unit: "rem" },
};
const store = (() => { try { return JSON.parse(localStorage.getItem("nabu-view") ?? "{}"); } catch { return {}; } })();
const save = () => { try { localStorage.setItem("nabu-view", JSON.stringify(store)); } catch {} };
const inputs = [...document.querySelectorAll(".settings input")];
const applyAll = () => {
  for (const input of inputs) {
    const key = input.dataset.key, s = SETTINGS[key];
    input.value = store[key] ?? s.def;
    s.apply(input.value);
    input.nextElementSibling.value = `${input.value}${s.unit}`;
  }
};
for (const input of inputs) input.addEventListener("input", () => { store[input.dataset.key] = Number(input.value); save(); applyAll(); });
// Theme: "" follows the OS; "light" / "dark" pin it. The <head> script applies it before first paint.
// The header button flips whatever is showing now and shows the theme it will switch to.
const themeButtons = [...$("s-theme").querySelectorAll("button")];
const themeBtn = $("theme-btn");
const osDark = matchMedia("(prefers-color-scheme: dark)");
const showing = () => store.theme || (osDark.matches ? "dark" : "light");
const ICONS = {
  dark: '<path d="M20 14.5A8 8 0 0 1 9.5 4a8 8 0 1 0 10.5 10.5z"/>',
  light: '<circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/>',
};
const applyTheme = () => {
  const t = store.theme ?? "";
  if (t) document.documentElement.dataset.theme = t; else delete document.documentElement.dataset.theme;
  for (const b of themeButtons) b.setAttribute("aria-pressed", String(b.dataset.theme === t));
  const next = showing() === "dark" ? "light" : "dark";
  themeBtn.innerHTML = `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${ICONS[next]}</svg>`;
  themeBtn.title = `${next} にする (t)`;
  themeBtn.setAttribute("aria-label", `${next} テーマにする`);
};
const setTheme = (t) => { if (t) store.theme = t; else delete store.theme; save(); applyTheme(); };
const flipTheme = () => setTheme(showing() === "dark" ? "light" : "dark");
for (const b of themeButtons) b.addEventListener("click", () => setTheme(b.dataset.theme));
themeBtn.addEventListener("click", flipTheme);
osDark.addEventListener("change", applyTheme);
$("s-reset").addEventListener("click", () => { for (const k of [...Object.keys(SETTINGS), "theme"]) delete store[k]; save(); applyAll(); applyTheme(); });
applyAll();
applyTheme();

// ---- narrow: sidebar as a popover drawer -------------------------------------
const side = $("side"), sideBtn = $("side-btn");
const narrow = matchMedia("(max-width: 48rem)");
const applyNarrow = () => { if (narrow.matches) side.setAttribute("popover", "auto"); else { if (side.matches(":popover-open")) side.hidePopover(); side.removeAttribute("popover"); } };
narrow.addEventListener("change", applyNarrow);
applyNarrow();
sideBtn.addEventListener("click", () => { side.togglePopover(); if (side.matches(":popover-open")) q.focus(); });
side.addEventListener("toggle", (e) => sideBtn.setAttribute("aria-expanded", String(e.newState === "open")));
const closeSide = () => { if (side.matches(":popover-open")) side.hidePopover(); };

// ---- boot -------------------------------------------------------------------
renderTree();
renderFacets();
main.tabIndex = -1;
q.addEventListener("input", filter);
window.addEventListener("hashchange", () => select(location.hash.slice(1), false));
select(location.hash.slice(1), false);
