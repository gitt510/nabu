const tasks = JSON.parse(document.getElementById("data").textContent);
const byId = Object.fromEntries(tasks.map((t) => [t.slug, t]));
const STATUS = ["doing", "inbox", "done"];
const $ = (id) => document.getElementById(id);
const tree = $("tree");
const doc = $("doc");
const q = $("q");
const main = document.querySelector(".main");

const fmt = (iso) => {
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString("ja-JP", { timeZone: "Asia/Tokyo", month: "numeric", day: "numeric", hour: "2-digit", minute: "2-digit" });
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

// ---- tree -----------------------------------------------------------------
function renderTree() {
  tree.replaceChildren();
  for (const st of STATUS) {
    const files = tasks.filter((t) => t.status === st);
    const dir = el("li", { class: `dir ${st}${st === "done" ? " closed" : ""}` });
    const name = el("div", { class: "dir-name", onclick: () => dir.classList.toggle("closed") }, `${st}/`, el("span", { class: "n" }, String(files.length)));
    const ul = el("ul", { class: "files" });
    for (const t of files) {
      const fm = t.frontmatter, flag = fm.waiting ? "⏳" : fm.scheduled ? fmt(fm.scheduled).split(" ")[0] : "";
      const b = el("button", { class: "file", "data-slug": t.slug, type: "button", tabindex: "-1", title: t.slug, onclick: () => select(t.slug) }, t.title);
      if (flag) b.append(el("span", { class: "flag" }, flag));
      ul.append(el("li", {}, b));
    }
    dir.append(name, ul);
    tree.append(dir);
  }
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
  const t = byId[slug];
  if (!t) return;
  for (const b of tree.querySelectorAll(".file")) b.classList.toggle("active", b.dataset.slug === slug);
  tree.querySelector(`.dir.${t.status}`).classList.remove("closed");
  const meta = el("dl", { class: "meta" });
  const fm = t.frontmatter;
  if (fm.scheduled) meta.append(el("div", {}, el("dt", {}, "期日"), el("dd", {}, fmt(fm.scheduled))));
  if (fm.waiting) meta.append(el("div", {}, el("dt", {}, "待ち"), el("dd", {}, fm.waiting)));
  if (fm.tickets?.length) meta.append(el("div", {}, el("dt", {}, "ticket"), el("dd", {}, ...fm.tickets.map((u) => el("a", { href: u, target: "_blank", rel: "noopener" }, ticket(u))))));
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
  for (const b of tree.querySelectorAll(".file")) b.parentElement.classList.toggle("hidden", !words.every((w) => byId[b.dataset.slug].text.includes(w)));
  for (const dir of tree.querySelectorAll(".dir")) {
    const n = dir.querySelectorAll("li:not(.hidden)").length;
    dir.querySelector(".n").textContent = String(n);
    dir.classList.toggle("hidden", n === 0);
    if (words.length) dir.classList.remove("closed");
  }
}

// ---- shortcuts (single keys, outside text fields; ? lists them) ------------
const inTextField = (e) => ["INPUT", "TEXTAREA", "SELECT"].includes(e.target.tagName) || e.target.isContentEditable;
const shortcuts = [
  { group: "移動" },
  { key: "/", label: "検索にフォーカス", run: () => q.focus() },
  { key: "h", label: "file tree にフォーカス", run: () => focusFile(0) },
  { key: "n", label: "次のタスクを開く", run: () => step(1) },
  { key: "p", label: "前のタスクを開く", run: () => step(-1) },
  { key: "j", label: "本文をスクロール", run: () => main.scrollBy({ top: 80 }) },
  { key: "k", label: "本文をスクロール", run: () => main.scrollBy({ top: -80 }) },
  { group: "tree 内" },
  { key: "j / ↓", label: "下へ", tree: true },
  { key: "k / ↑", label: "上へ", tree: true },
  { key: "gg / G", label: "先頭 / 末尾", tree: true },
  { key: "l / Enter", label: "開く（移動しただけでも切り替わる）", tree: true },
  { key: "Esc", label: "tree から抜ける", tree: true },
  { group: "その他" },
  { key: "?", label: "この一覧", run: () => toggle($("help")) },
  { key: ",", label: "設定", run: () => toggle($("settings")) },
  { key: "Esc", label: "検索を消す / dialog を閉じる", tree: true },
];
const toggle = (d) => (d.open ? d.close() : d.showModal());
let pendingG = false;

document.addEventListener("keydown", (e) => {
  if (e.isComposing || e.metaKey || e.ctrlKey || e.altKey) return;
  if (document.querySelector("dialog[open]")) return; // dialog handles Esc itself
  if (inTextField(e)) {
    if (e.key === "Escape") { if (q.value) { q.value = ""; filter(); } else focusFile(0); }
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
$("help-btn").addEventListener("click", () => toggle($("help")));
$("settings-btn").addEventListener("click", () => toggle($("settings")));

const SETTINGS = {
  font: { def: 16, apply: (v) => (document.documentElement.style.fontSize = `${v}px`), unit: "px" },
  side: { def: 300, apply: (v) => document.documentElement.style.setProperty("--side-w", `${v}px`), unit: "px" },
  width: { def: 46, apply: (v) => document.documentElement.style.setProperty("--doc-w", `${v}rem`), unit: "rem" },
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
main.tabIndex = -1;
q.addEventListener("input", filter);
window.addEventListener("hashchange", () => select(location.hash.slice(1), false));
const first = location.hash.slice(1) || tasks.find((t) => t.status === "doing")?.slug || tasks[0]?.slug;
if (first) select(first, false);
