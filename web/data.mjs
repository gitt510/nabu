// Builds dist/tasks.json, the data the viewer fetches, from data.json, the output of `nabu task ls --json`.
import { execFileSync } from "node:child_process";
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { homedir } from "node:os";
import { marked } from "marked";

const here = (name) => new URL(name, import.meta.url);

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

const tasks = JSON.parse(readFileSync(here("data.json"), "utf8")).map((r) => {
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

mkdirSync(here("dist/"), { recursive: true });
writeFileSync(here("dist/tasks.json"), JSON.stringify(tasks));
