import { marked } from "marked";

import { type Habit, SECTIONS, type Task } from "../api/schema";

/** 本文の 1 section：見出しと、行を箇条書きにした HTML（行がなければ空） */
export type Part = { key: (typeof SECTIONS)[number]; title: string; html: string };

/** page が見せる形の task：本文は Why と DoD をいつも、Why dropped は行があるときだけ */
export type Item = Task & { parts: Part[] };

const HEADINGS: Record<(typeof SECTIONS)[number], string> = {
  why: "Why",
  dod: "DoD",
  why_dropped: "Why dropped",
};

const toItem = (t: Task): Item => {
  const lists = SECTIONS.filter((k) => k !== "why_dropped" || t[k].length).map((k) => ({
    key: k,
    title: HEADINGS[k],
    md: t[k].map((l) => `- ${l}`).join("\n"),
  }));
  const parts = lists.map(({ key, title, md }) => ({
    key,
    title,
    html: md && marked.parse(md, { async: false }),
  }));
  return { ...t, parts };
};

/** 全部の task を 1 回で読む。no-store なので、reload すると最新の書き込みが見える */
export async function fetchTasks(): Promise<Item[]> {
  const res = await fetch("/api/tasks", { cache: "no-store" });
  if (!res.ok) throw new Error(`Couldn't load tasks (${res.status})`);
  return ((await res.json()) as Task[]).map(toItem);
}

/** task を書き換える。断られたら理由を返す */
export const patch = (id: number, fields: Partial<Task>) =>
  send("PATCH", `/api/tasks/${id}`, fields);

/** task を done にする、または todo に戻す。どちらも dropped の理由は消す */
export const check = (id: number, done: boolean) =>
  patch(id, { status: done ? "done" : "todo", why_dropped: [] });

/** 全部の habit を、day（YYYY-MM-DD）にやったかどうか付きで読む */
export async function fetchHabits(day: string): Promise<Habit[]> {
  const res = await fetch(`/api/habits?date=${day}`, { cache: "no-store" });
  if (!res.ok) throw new Error(`Couldn't load habits (${res.status})`);
  return (await res.json()) as Habit[];
}

export const addHabit = (title: string) => send("POST", "/api/habits", { title });
export const renameHabit = (id: number, title: string) =>
  send("PATCH", `/api/habits/${id}`, { title });
export const deleteHabit = (id: number) => send("DELETE", `/api/habits/${id}`);
/** day にやった印を付ける、または外す */
export const checkHabit = (id: number, day: string, done: boolean) =>
  send(done ? "PUT" : "DELETE", `/api/habits/${id}/checks/${day}`);

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
  if (res.type === "opaqueredirect" || res.status === 401)
    return "Your session expired. Reload the page and sign in again.";
  if (!res.ok)
    return (
      ((await res.json().catch(() => ({}))) as { error?: string }).error ??
      `Couldn't save (${res.status})`
    );
}
