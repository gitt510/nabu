import { marked } from "marked";

import { SECTIONS, type Task } from "../src/fields";

/** page が見せる形の task：本文は題の下の HTML、検索は小文字の全文 */
export type Item = Task & { html: string; text: string };

const heading = (k: string) => k[0].toUpperCase() + k.slice(1).replace("_", " ");

const toItem = (t: Task): Item => {
  // 空の理由は出さない。nabu task read と同じ
  const body = SECTIONS.filter((k) => k !== "why_dropped" || t.why_dropped.length)
    .map((k) => `### ${heading(k)}\n\n${t[k].map((l) => `- ${l}`).join("\n")}`)
    .join("\n\n");
  return { ...t, html: marked.parse(body, { async: false }), text: [t.title, t.slug, t.waiting ?? "", t.project ?? "", body].join("\n").toLowerCase() };
};

/** 全部の task を 1 回で読む。no-store なので、reload すると最新の書き込みが見える */
export async function fetchTasks(): Promise<Item[]> {
  const res = await fetch("/api/tasks", { cache: "no-store" });
  if (!res.ok) throw new Error(`Couldn't load tasks (${res.status})`);
  return ((await res.json()) as Task[]).map(toItem);
}

/** task を書き換える。断られたら理由を返す */
export async function patch(slug: string, fields: Partial<Task>): Promise<string | undefined> {
  let res: Response;
  try {
    res = await fetch(`/api/tasks/${encodeURIComponent(slug)}`, {
      method: "PATCH",
      headers: { "content-type": "application/json" },
      body: JSON.stringify(fields),
      redirect: "manual",
    });
  } catch {
    return "Couldn't save. Check your connection and try again.";
  }
  // Access は切れた session に sign-in page への redirect で答える
  if (res.type === "opaqueredirect" || res.status === 401) return "Your session expired. Reload the page and sign in again.";
  if (!res.ok) return ((await res.json().catch(() => ({}))) as { error?: string }).error ?? `Couldn't save (${res.status})`;
}
