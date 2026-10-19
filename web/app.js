import { marked } from "./marked.js";

// AREAS, STATUSES, LISTS, and SECTIONS come from web/src/schema.ts: build.mjs writes them above this file.

// Every task comes from the API in one request, and the page narrows them in memory; no-store, so a reload
// shows the latest write. Each task gains what the page shows: its frontmatter as `nabu task read` prints it,
// the body as HTML under the title, and a lowercase search text.
const tasks = (await (await fetch("api/tasks", { cache: "no-store" })).json())
  .map((r) => {
    const frontmatter = { created: r.created, area: r.area };
    for (const k of ["project", "scheduled", "waiting"]) if (r[k]) frontmatter[k] = r[k];
    for (const k of LISTS) if (r[k].length) frontmatter[k] = r[k];
    // an empty reason is left out, as nabu task read leaves it out
    const body = SECTIONS.filter(([, k]) => k !== "why_dropped" || r.why_dropped.length)
      .map(([h, k]) => `## ${h}\n\n${r[k].map((l) => `- ${l}`).join("\n")}`).join("\n\n");
    return {
      slug: r.slug,
      status: r.status,
      title: r.title,
      frontmatter,
      updated: r.updated,
      html: marked.parse(body),
      text: [r.title, r.slug, frontmatter.waiting ?? "", frontmatter.project ?? "", body].join("\n").toLowerCase(),
    };
  })
  .sort((a, b) => (a.slug < b.slug ? -1 : a.slug > b.slug ? 1 : 0)); // the views order; this only fixes ties
const byId = Object.fromEntries(tasks.map((t) => [t.slug, t]));
const STATUS = ["doing", "inbox", "done", "dropped"];
const $ = (id) => document.getElementById(id);
const files = $("files");
const doc = $("doc");
const q = $("q");
const main = document.querySelector(".main");
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

// ---- views -----------------------------------------------------------------
// The first column, like Finder's sidebar: one view per frontmatter key, then one per status folder,
// then the area and project filters. Every item carries a data-key so focus survives a re-render.
// A task may sit in two views (a doing task with a date is in Scheduled and in Doing).
const open_ = (t) => t.status !== "done" && t.status !== "dropped";
const bySchedule = (a, b) => instant(a.frontmatter.scheduled) - instant(b.frontmatter.scheduled);
const svg = (d) => `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${d}</svg>`;
const VIEWS = [
  { id: "all", label: "All", pick: () => true, icon: '<rect x="3" y="3" width="7" height="7" rx="1.5"/><rect x="14" y="3" width="7" height="7" rx="1.5"/><rect x="3" y="14" width="7" height="7" rx="1.5"/><rect x="14" y="14" width="7" height="7" rx="1.5"/>' },
  { id: "scheduled", label: "Scheduled", pick: (t) => open_(t) && t.frontmatter.scheduled, icon: '<rect x="3" y="5" width="18" height="16" rx="2"/><path d="M3 10h18M8 3v4M16 3v4"/>' },
  { id: "waiting", label: "Waiting", pick: (t) => open_(t) && t.frontmatter.waiting, icon: '<path d="M6 3h12M6 21h12M8 3v4a4 4 0 0 0 8 0V3M8 21v-4a4 4 0 0 1 8 0v4"/>' },
  { id: "doing", label: "Doing", pick: (t) => t.status === "doing", folder: true, icon: '<circle cx="12" cy="12" r="9"/><path d="m10 8 5 4-5 4z"/>' },
  { id: "inbox", label: "Inbox", pick: (t) => t.status === "inbox", folder: true, icon: '<path d="M3 13h5l2 3h4l2-3h5"/><path d="M5 4h14l2 9v7H3v-7z"/>' },
  { id: "done", label: "Done", pick: (t) => t.status === "done", folder: true, icon: '<circle cx="12" cy="12" r="9"/><path d="m8 12 3 3 5-6"/>' },
  { id: "dropped", label: "Dropped", pick: (t) => t.status === "dropped", folder: true, icon: '<circle cx="12" cy="12" r="9"/><path d="m9 9 6 6M15 9l-6 6"/>' },
];
const PROPS = ["area", "project", "created", "updated", "tickets", "prs", "links"];
const today = new Date().toLocaleDateString("ja-JP", { timeZone: "Asia/Tokyo" });
const dayOf = (iso) => instant(iso).toLocaleDateString("ja-JP", { timeZone: "Asia/Tokyo" });
const isLate = (iso) => instant(iso) < new Date() && dayOf(iso) !== today;

const store = (() => { try { return JSON.parse(localStorage.getItem("nabu-view") ?? "{}"); } catch { return {}; } })();
const save = () => { try { localStorage.setItem("nabu-view", JSON.stringify(store)); } catch {} };
const viewOf = (id) => VIEWS.find((v) => v.id === id) ?? VIEWS[0];
let view = viewOf(store.view);

// What the search box and the area / project rows let through.
function keep(t) {
  const words = q.value.trim().toLowerCase().split(/\s+/).filter(Boolean);
  return words.every((w) => t.text.includes(w)) && (!pick.area || t.frontmatter.area === pick.area) && (!pick.project || t.frontmatter.project === pick.project);
}
// The list's order: by when each task was filed, newest first unless the header's arrow turned it.
let oldest = store.oldest === true;
const listed = (v) => {
  const sign = oldest ? 1 : -1;
  return tasks.filter((t) => v.pick(t) && keep(t)).sort((a, b) => sign * (instant(a.frontmatter.created) - instant(b.frontmatter.created)));
};
function flipOrder() {
  oldest = !oldest;
  store.oldest = oldest;
  save();
  renderList();
}

function fileButton(t) {
  const fm = t.frontmatter;
  const day = (iso) => fmt(iso).split(" ")[0];
  // One line: the title, a flag while it waits on someone else, and when it was filed, so a task left alone shows.
  return el("button", { class: "file", "data-slug": t.slug, "data-status": t.status, type: "button", tabindex: "-1", title: t.slug, onclick: () => select(t.slug) },
    el("span", { class: "name" }, t.title), fm.waiting ? el("span", { class: "flag" }, "⏳") : "", el("span", { class: "when" }, day(fm.created)));
}

// Area, then the projects in it. Each is a single choice; pressing the chosen one again lets everything through.
function renderViews() {
  const row = (label, on, onclick, icon, ...rest) => {
    const b = el("button", { class: "view", type: "button", tabindex: on ? "0" : "-1", "data-key": label, "aria-current": String(on), onclick });
    b.innerHTML = icon;
    b.append(el("span", {}, label), ...rest);
    return b;
  };
  const head = (s) => el("div", { class: "views-h" }, s);
  const chip = (kind, label, on, onclick) => el("button", { class: `chip ${kind}`, type: "button", tabindex: "-1", "data-key": `${kind}:${label}`, "aria-pressed": String(on), onclick }, label);
  const viewRow = (v) => row(v.label, v.id === view.id, () => setView(v), svg(v.icon), el("span", { class: "n" }, String(listed(v).length)));
  const projects = [...new Set(tasks.filter((t) => !pick.area || t.frontmatter.area === pick.area).map((t) => t.frontmatter.project).filter(Boolean))].sort();
  const focused = $("views").contains(document.activeElement) ? document.activeElement.dataset.key : null;
  $("views").replaceChildren(
    head("Views"), ...VIEWS.filter((v) => !v.folder).map(viewRow),
    head("Status"), ...VIEWS.filter((v) => v.folder).map(viewRow),
    head("Area"), el("div", { class: "chips" }, ...AREAS.map((a) => chip("area", a, pick.area === a, () => narrowTo(pick.area === a ? "" : a, "")))),
    ...(projects.length ? [head("Project"), el("div", { class: "chips" }, ...projects.map((p) => chip("project", p, pick.project === p, () => narrowTo(pick.area, pick.project === p ? "" : p))))] : []),
  );
  if (focused) ($("views").querySelector(`[data-key="${CSS.escape(focused)}"]`) ?? $("views").querySelector("button")).focus();
}
function narrowTo(area, project) {
  pick.area = area;
  pick.project = project;
  filter();
}

function renderList() {
  const active = files.querySelector(".file.active")?.dataset.slug ?? location.hash.slice(1);
  const items = listed(view);
  files.replaceChildren(...(items.length ? items.map((t) => el("li", {}, fileButton(t))) : [el("li", { class: "empty" }, "なし")]));
  const stop = files.querySelector(`.file[data-slug="${CSS.escape(active)}"]`) ?? files.querySelector(".file");
  stop?.classList.toggle("active", stop.dataset.slug === active);
  if (stop) stop.tabIndex = 0; // the column's one Tab stop
  $("count").textContent = `${items.length} / ${tasks.filter(view.pick).length}`;
  const order = $("order");
  order.innerHTML = `created${svg(oldest ? '<path d="M12 19V5M5 12l7-7 7 7"/>' : '<path d="M12 5v14M5 12l7 7 7-7"/>')}`;
  order.title = `${oldest ? "古い" : "新しい"}順 (s)`;
}

function setView(v) {
  view = v;
  store.view = v.id;
  save();
  renderViews();
  renderList();
}
const stepView = (delta) => setView(VIEWS[(VIEWS.indexOf(view) + delta + VIEWS.length) % VIEWS.length]);

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

const visibleFiles = () => [...files.querySelectorAll(".file")];
const views = $("views");
const sideItems = () => [...views.querySelectorAll("button")];

// Move the focus inside the sidebar. Landing on a view opens it (as in Finder); a chip waits for Enter.
function focusSide(delta) {
  const items = sideItems();
  const at = items.indexOf(document.activeElement);
  let next;
  if (at < 0) next = Math.max(0, items.findIndex((b) => b.getAttribute("aria-current") === "true"));
  else next = Math.min(items.length - 1, Math.max(0, at + delta));
  items[next].focus();
  items[next].scrollIntoView({ block: "nearest" });
  if (items[next].classList.contains("view") && items[next].getAttribute("aria-current") !== "true") items[next].click(); // re-renders; the focus follows the key
}

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
  if (slug === "") {
    for (const b of files.querySelectorAll(".file")) b.classList.remove("active");
    renderHome();
    if (push) history.replaceState(null, "", location.pathname);
    main.scrollTop = 0;
    if (push) closeSide();
    return;
  }
  const t = byId[slug];
  if (!t) return;
  // A task outside the current view (opened from the URL or Home) shows under All.
  if (!view.pick(t) && view.id !== "all") setView(VIEWS[0]);
  for (const b of files.querySelectorAll(".file")) b.classList.toggle("active", b.dataset.slug === slug);
  // Properties: every key in PROPS order, always, so each sits in the same place; a key without a value shows a dash.
  const meta = el("dl", { class: "meta" });
  const fm = { ...t.frontmatter, updated: t.updated };
  const value = (key, v) => {
    if (Array.isArray(v)) return v.map((u) => el("a", { href: u, target: "_blank", rel: "noopener" }, ticket(u)));
    if (key === "created" || key === "updated" || key === "scheduled") return [fmt(v)];
    if (key === "area" || key === "project") return [el("button", { class: `chip ${key}`, type: "button", title: `${v} で絞り込む`, onclick: () => narrowTo(fm.area, key === "project" ? v : "") }, v)];
    return [v];
  };
  for (const key of PROPS) meta.append(el("div", {}, el("dt", {}, key), el("dd", {}, ...(fm[key]?.length ? value(key, fm[key]) : [el("span", { class: "none" }, "—")]))));
  const body = el("div", { class: "body" });
  body.innerHTML = t.html;
  // The slug and the title stay on top while the body scrolls; the rail sticks just under them.
  const head = el("header", { class: "doc-head" }, el("div", { class: "crumb" }, crumb(t), statusControl(t)), el("p", { class: "error", id: "doc-error", role: "alert" }), el("h1", {}, t.title));
  doc.replaceChildren(head, el("div", { class: "railed" }, body, meta));
  headSize.disconnect();
  headSize.observe(head);
  if (push) history.replaceState(null, "", `#${slug}`);
  main.scrollTop = 0;
  if (push) closeSide();
}

const headSize = new ResizeObserver(([e]) => doc.style.setProperty("--head-h", `${e.target.offsetHeight}px`));

// The slug line above the title; a click copies the slug, the name every nabu task command takes.
function crumb(t) {
  const b = el("button", { class: "copy", type: "button", title: `${t.slug}\nclick で copy` }, t.slug);
  b.addEventListener("click", () => navigator.clipboard.writeText(t.slug).then(() => {
    b.textContent = "copied";
    setTimeout(() => { b.textContent = t.slug; }, 1200);
  }));
  return b;
}

// ---- writes ----------------------------------------------------------------
// The page changes a task through the API and then reloads, so the API stays the one source of what it shows.
// patch returns why the API refused, or reloads on success.
async function patch(slug, fields) {
  let res;
  try {
    res = await fetch(`api/tasks/${encodeURIComponent(slug)}`, {
      method: "PATCH", headers: { "content-type": "application/json" }, body: JSON.stringify(fields), redirect: "manual",
    });
  } catch { return "保存できませんでした。通信を確かめてください"; }
  // Access answers an expired session with a redirect to its sign-in page.
  if (res.type === "opaqueredirect" || res.status === 401) return "session が切れました。reload して sign in し直してください";
  if (!res.ok) return (await res.json().catch(() => ({}))).error ?? `保存できませんでした (${res.status})`;
  location.reload();
}

// The status of the open task: a button that opens a menu of the statuses, each with its sidebar icon.
// Dropped sits apart below a rule and asks why first. The menu follows the ARIA menu pattern: arrows,
// Home / End, 1–4, Enter; Esc and a click outside close it (popover="auto").
const statusLabel = (s) => (s === "dropped" ? "Drop…" : s[0].toUpperCase() + s.slice(1));
const statusIcon = (s) => svg(VIEWS.find((v) => v.id === s).icon);
const chevron = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m6 9 6 6 6-6"/></svg>';
const check = '<svg class="check" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m5 12 5 5 9-10"/></svg>';

function statusControl(t) {
  const trigger = el("button", { type: "button", class: "status-btn", popovertarget: "status-menu", "aria-haspopup": "menu", "aria-expanded": "false", title: "status を変える (m)" });
  trigger.innerHTML = statusIcon(t.status) + `<span>${t.status === "dropped" ? "Dropped" : statusLabel(t.status)}</span>` + chevron;
  const menu = el("div", { id: "status-menu", class: "menu", popover: "auto", role: "menu", "aria-label": "status" });
  STATUSES.forEach((s, i) => {
    if (s === "dropped") menu.append(el("div", { class: "sep", role: "separator" }));
    const item = el("button", { type: "button", role: "menuitemradio", "aria-checked": String(t.status === s), tabindex: "-1", onclick: () => { menu.hidePopover(); move(t, s); } });
    item.innerHTML = statusIcon(s) + `<span class="label">${statusLabel(s)}</span>` + (t.status === s ? check : "") + `<span class="key">${i + 1}</span>`;
    menu.append(item);
  });
  const items = () => [...menu.querySelectorAll("[role=menuitemradio]")];
  const focusAt = (i) => { const all = items(); all[(i + all.length) % all.length].focus(); };
  // Below the button, its right edge on the button's; above it when the page has no room below.
  menu.addEventListener("beforetoggle", (e) => {
    if (e.newState !== "open") return;
    const r = trigger.getBoundingClientRect();
    Object.assign(menu.style, { top: `${r.bottom + 4}px`, right: `${document.documentElement.clientWidth - r.right}px` });
  });
  menu.addEventListener("toggle", (e) => {
    const open = e.newState === "open";
    trigger.setAttribute("aria-expanded", String(open));
    if (open) {
      const r = trigger.getBoundingClientRect();
      const h = menu.getBoundingClientRect().height;
      if (r.bottom + 4 + h > innerHeight && r.top - 4 - h > 0) menu.style.top = `${r.top - 4 - h}px`;
      focusAt(Math.max(0, STATUSES.indexOf(t.status)));
    } else if (!document.activeElement || document.activeElement === document.body || menu.contains(document.activeElement)) trigger.focus();
  });
  // Every key inside the menu stays there, so the page's shortcuts never fire behind it.
  menu.addEventListener("keydown", (e) => {
    e.stopPropagation();
    const at = items().indexOf(document.activeElement);
    if (e.key === "ArrowDown" || e.key === "j") focusAt(at + 1);
    else if (e.key === "ArrowUp" || e.key === "k") focusAt(at - 1);
    else if (e.key === "Home") focusAt(0);
    else if (e.key === "End") focusAt(-1);
    else if (/^[1-4]$/.test(e.key)) items()[Number(e.key) - 1].click();
    else if (e.key === "Tab") menu.hidePopover();
    else return;
    e.preventDefault();
  });
  return el("span", { class: "status-picker" }, trigger, menu);
}
async function move(t, status) {
  if (t.status === status) return;
  if (status === "dropped") {
    $("drop").dataset.slug = t.slug;
    $("drop-why").value = "";
    $("drop-error").textContent = "";
    return $("drop").showModal();
  }
  $("doc-error").textContent = (await patch(t.slug, { status })) ?? "";
}
$("drop-form").addEventListener("submit", async (e) => {
  e.preventDefault();
  const why = $("drop-why").value.split("\n").map((l) => l.trim()).filter(Boolean);
  if (!why.length) return ($("drop-error").textContent = "理由を 1 行以上書いてください");
  $("drop-error").textContent = (await patch($("drop").dataset.slug, { status: "dropped", why_dropped: why })) ?? "";
});

// Open the task adjacent to the current one, regardless of focus.
function step(delta) {
  const items = visibleFiles();
  const at = items.findIndex((b) => b.classList.contains("active"));
  const next = items[Math.min(items.length - 1, Math.max(0, at + delta))];
  if (next) { select(next.dataset.slug); next.scrollIntoView({ block: "nearest" }); if (files.contains(document.activeElement)) next.focus(); }
}

function filter() {
  renderViews();
  renderList();
}

// ---- shortcuts (single keys, outside text fields; ? lists them) ------------
// Three columns: sidebar, list, document. ← → move the focus between them (→ from the sidebar lands on
// the top of the list); Tab / Shift+Tab do the same with one stop per column. ↑ ↓ walk the sidebar,
// pick a file in the list, and scroll the document.
const inTextField = (e) => ["INPUT", "TEXTAREA", "SELECT"].includes(e.target.tagName) || e.target.isContentEditable;
const scroll = (dy) => main.scrollBy({ top: dy });
const shortcuts = [
  { group: "移動" },
  { key: "/", label: "検索にフォーカス", run: () => q.focus() },
  { key: "← → / Tab", label: "列を移る（サイドバー → 一覧 → 本文）", run: null },
  { key: "h", run: () => focusFile(0) },
  { key: "ArrowLeft", run: () => focusFile(0) },
  { key: "H", label: "Home を開く", run: () => select("") },
  { key: "n", label: "次のタスクを開く", run: () => step(1) },
  { key: "p", label: "前のタスクを開く", run: () => step(-1) },
  { key: "[ / ]", label: "前 / 次の一覧へ", run: null },
  { key: "[", run: () => stepView(-1) },
  { key: "]", run: () => stepView(1) },
  { key: "s", label: "created の新しい順と古い順を入れ替える", run: flipOrder },
  { key: "m", label: "開いているタスクの status を変える", run: () => $("status-menu")?.togglePopover() },
  { key: "j / ↓", label: "本文を下へスクロール", run: null },
  { key: "j", run: () => scroll(80) },
  { key: "ArrowDown", run: () => scroll(80) },
  { key: "k / ↑", label: "本文を上へスクロール", run: null },
  { key: "k", run: () => scroll(-80) },
  { key: "ArrowUp", run: () => scroll(-80) },
  { group: "一覧内" },
  { key: "j / ↓", label: "下へ（移動した先が開く）", tree: true },
  { key: "k / ↑", label: "上へ", tree: true },
  { key: "gg / G", label: "先頭 / 末尾", tree: true },
  { key: "Enter", label: "開く", tree: true },
  { key: "l / →", label: "本文にフォーカス", tree: true },
  { key: "h / ←", label: "サイドバーにフォーカス", tree: true },
  { key: "Esc", label: "本文にフォーカス", tree: true },
  { group: "サイドバー内" },
  { key: "j / ↓", label: "下へ（view は移動した先に切り替わる）", tree: true },
  { key: "k / ↑", label: "上へ", tree: true },
  { key: "Enter", label: "area / project を絞り込む・解除する", tree: true },
  { key: "l / →", label: "一覧の先頭にフォーカス", tree: true },
  { key: "Esc", label: "本文にフォーカス", tree: true },
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
  if (files.contains(e.target)) {
    const k = e.key;
    if (k === "Tab") return;
    if (k === "g") { if (pendingG) { focusFile("first"); pendingG = false; } else pendingG = true; e.preventDefault(); return; }
    pendingG = false;
    if (k === "j" || k === "ArrowDown") focusFile(1);
    else if (k === "k" || k === "ArrowUp") focusFile(-1);
    else if (k === "G") focusFile("last");
    else if (k === "Enter") select(e.target.dataset.slug);
    else if (k === "l" || k === "ArrowRight" || k === "Escape") main.focus();
    else if (k === "h" || k === "ArrowLeft") focusSide(0);
    else { pendingG = false; return runGlobal(e); }
    e.preventDefault();
    return;
  }
  if (views.contains(e.target)) {
    const k = e.key;
    if (k === "j" || k === "ArrowDown") focusSide(1);
    else if (k === "k" || k === "ArrowUp") focusSide(-1);
    else if (k === "l" || k === "ArrowRight") focusFile("first");
    else if (k === "Escape") main.focus();
    else if (k === "Enter" || k === " " || k === "Tab") return; // the button activates itself
    else return runGlobal(e);
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
$("keys").append(...shortcuts.flatMap((s) => (s.group ? [el("div", { class: "group" }, s.group)] : s.label ? [el("dt", {}, s.key), el("dd", {}, s.label)] : [])));
for (const d of document.querySelectorAll("dialog")) d.querySelector(".close").addEventListener("click", () => d.close());
$("home-btn").addEventListener("click", () => select(""));
$("order").addEventListener("click", flipOrder);
$("help-btn").addEventListener("click", () => toggle($("help")));
$("settings-btn").addEventListener("click", () => toggle($("settings")));

const SETTINGS = {
  font: { def: 18, apply: (v) => (document.documentElement.style.fontSize = `${v}px`), unit: "px" },
  side: { def: 264, apply: (v) => document.documentElement.style.setProperty("--side-w", `${v}px`), unit: "px" },
  width: { def: 50, apply: (v) => document.documentElement.style.setProperty("--doc-w", `${v}rem`), unit: "rem" },
};
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
  document.documentElement.classList.toggle("dark", showing() === "dark");
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
// Narrow: the drawer takes the file list along, so one tap reaches both the views and the files.
const applyNarrow = () => {
  if (narrow.matches) { side.setAttribute("popover", "auto"); side.append($("list")); }
  else { if (side.matches(":popover-open")) side.hidePopover(); side.removeAttribute("popover"); document.querySelector(".content").prepend($("list")); }
};
narrow.addEventListener("change", applyNarrow);
applyNarrow();
sideBtn.addEventListener("click", () => { side.togglePopover(); if (side.matches(":popover-open")) q.focus(); });
side.addEventListener("toggle", (e) => sideBtn.setAttribute("aria-expanded", String(e.newState === "open")));
const closeSide = () => { if (side.matches(":popover-open")) side.hidePopover(); };

// ---- boot -------------------------------------------------------------------
renderViews();
renderList();
main.tabIndex = 0;
q.addEventListener("input", filter);
window.addEventListener("hashchange", () => select(location.hash.slice(1), false));
select(location.hash.slice(1), false);
