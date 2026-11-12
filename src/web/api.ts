import { marked } from "marked";

import type { Habit, Item, Task } from "../api/schema";

/** page が見せる形の task：やめた理由があれば、箇条書きにした HTML を添える */
export type TaskView = Task & { whyDropped: string | null };

const toTaskView = (t: Task): TaskView => ({
  ...t,
  whyDropped: t.why_dropped.length
    ? marked.parse(t.why_dropped.map((l) => `- ${l}`).join("\n"), { async: false })
    : null,
});

/** 全部の task を 1 回で読む。no-store なので、reload すると最新の書き込みが見える */
export async function fetchTasks(): Promise<TaskView[]> {
  const res = await fetch("/api/tasks", { cache: "no-store" });
  if (!res.ok) throw new Error(`Couldn't load tasks (${res.status})`);
  return ((await res.json()) as Task[]).map(toTaskView);
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

/** task の checklist を、並びの順に読む */
export async function fetchItems(taskId: number): Promise<Item[]> {
  const res = await fetch(`/api/items?task_id=${taskId}`, { cache: "no-store" });
  if (!res.ok) throw new Error(`Couldn't load the checklist (${res.status})`);
  return (await res.json()) as Item[];
}

/** checklist の最後に行を足す */
export const addItem = (taskId: number, text: string) =>
  send("POST", "/api/items", { task_id: taskId, text });
export const patchItem = (id: number, fields: Partial<Item>) =>
  send("PATCH", `/api/items/${id}`, fields);
export const deleteItem = (id: number) => send("DELETE", `/api/items/${id}`);

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
