import { marked } from "marked";

import { type Habit, SECTIONS, type Task } from "../api/schema";

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
export const patch = (slug: string, fields: Partial<Task>) => send("PATCH", `/api/tasks/${encodeURIComponent(slug)}`, fields);

/** 全部の habit を、day（YYYY-MM-DD）にやったかどうか付きで読む */
export async function fetchHabits(day: string): Promise<Habit[]> {
  const res = await fetch(`/api/habits?date=${day}`, { cache: "no-store" });
  if (!res.ok) throw new Error(`Couldn't load habits (${res.status})`);
  return (await res.json()) as Habit[];
}

export const addHabit = (title: string) => send("POST", "/api/habits", { title });
export const renameHabit = (id: number, title: string) => send("PATCH", `/api/habits/${id}`, { title });
export const deleteHabit = (id: number) => send("DELETE", `/api/habits/${id}`);
/** day にやった印を付ける、または外す */
export const checkHabit = (id: number, day: string, done: boolean) => send(done ? "PUT" : "DELETE", `/api/habits/${id}/checks/${day}`);

/** API に書き込む。断られたら理由を返す */
async function send(method: string, path: string, body?: unknown): Promise<string | undefined> {
  let res: Response;
  try {
    res = await fetch(path, {
      method,
      headers: body === undefined ? undefined : { "content-type": "application/json" },
      body: body === undefined ? undefined : JSON.stringify(body),
      redirect: "manual",
    });
  } catch {
    return "Couldn't save. Check your connection and try again.";
  }
  // Access は切れた session に sign-in page への redirect で答える
  if (res.type === "opaqueredirect" || res.status === 401) return "Your session expired. Reload the page and sign in again.";
  if (!res.ok) return ((await res.json().catch(() => ({}))) as { error?: string }).error ?? `Couldn't save (${res.status})`;
}
