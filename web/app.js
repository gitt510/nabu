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
  doc.replaceChildren(el("p", { class: "crumb" }, today), el("h1", {}, "Today"), list(due, "今日のタスクはない"));
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
  const meta = el("dl", { class: "meta" });
  const fm = t.frontmatter;
  if (fm.scheduled) meta.append(el("div", {}, el("dt", {}, "期日"), el("dd", {}, fmt(fm.scheduled))));
  if (fm.waiting) meta.append(el("div", {}, el("dt", {}, "待ち"), el("dd", {}, fm.waiting)));
  for (const key of ["area", "project"]) if (fm[key]) meta.append(el("div", {}, el("dt", {}, key), el("dd", {}, el("button", { class: "chip", type: "button", title: `${fm[key]} で絞り込む`, onclick: () => narrowTo(fm.area, key === "project" ? fm.project : "") }, fm[key]))));
  for (const [key, label] of [["tickets", "ticket"], ["prs", "pr"], ["links", "link"]]) {
    if (fm[key]?.length) meta.append(el("div", {}, el("dt", {}, label), el("dd", {}, ...fm[key].map((u) => el("a", { href: u, target: "_blank", rel: "noopener" }, ticket(u))))));
  }
  const body = el("div", { class: "body" });
  body.innerHTML = t.html;
  doc.replaceChildren(el("p", { class: "crumb" }, t.path), el("h1", {}, t.title), meta, body);
  if (push) history.replaceState(null, "", `#${slug}`);
  main.scrollTop = 0;
  if (push) closeSide();
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
$("s-reset").addEventListener("click", () => { for (const k of Object.keys(SETTINGS)) delete store[k]; save(); applyAll(); });
applyAll();

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
