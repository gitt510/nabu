// Builds dist/tasks.json, the data the viewer fetches, from data.json, the output of `nabu task ls --json`.
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { marked } from "marked";

const here = (name) => new URL(name, import.meta.url);

const tasks = JSON.parse(readFileSync(here("data.json"), "utf8")).map((r) => {
  const frontmatter = r.frontmatter ?? {};
  let body = r.body.trim();
  if (body.startsWith("# " + r.title)) body = body.slice(r.title.length + 2);
  body = body.trim();
  return {
    path: r.path,
    file: r.path, // what the crumb copies, relative to the notebook
    slug: r.slug,
    status: r.status,
    title: r.title,
    frontmatter,
    updated: r.updated ?? "",
    html: marked.parse(body),
    text: [r.title, r.slug, frontmatter.waiting ?? "", frontmatter.project ?? "", body].join("\n").toLowerCase(),
  };
});
tasks.sort((a, b) => (a.slug < b.slug ? -1 : a.slug > b.slug ? 1 : 0)); // the viewer orders; this only fixes ties

mkdirSync(here("dist/"), { recursive: true });
writeFileSync(here("dist/tasks.json"), JSON.stringify(tasks));
