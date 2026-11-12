// The task contract: what /api/tasks accepts and returns. Every rule a write must meet lives here, so a task
// out of shape cannot reach the database, and the page reads its fields from here as the Worker does.
import { z } from "zod";

export const STATUSES = ["todo", "done", "dropped"] as const;
export const AREAS = ["work", "personal"] as const;

// LINE_LIMIT is the most characters a section line may carry: one fact a reader takes in at a glance.
export const LINE_LIMIT = 45;

// SECTIONS are the lists of lines a task carries: the reason a dropped task was closed, then the memo its
// owner adds while working on it.
export const SECTIONS = ["why_dropped", "memo"] as const;
export const LISTS = ["tickets", "prs", "links"] as const;

// The ids name the schemas the OpenAPI document shares as components, so a generated client names its types.
export const Status = z.enum(STATUSES).meta({ id: "Status" });
export type Status = z.output<typeof Status>;
export const Area = z.enum(AREAS).meta({ id: "Area" });

const kebab = z.string().regex(/^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$/, "must be lowercase kebab-case");

// scheduled is the planned work time: RFC3339 with an offset, or a date for the whole day. The pattern
// carries the shape into the JSON Schema; the refine still rejects a date that does not exist.
const scheduled = z
  .string()
  .regex(
    /^\d{4}-\d{2}-\d{2}(T\d{2}:\d{2}(:\d{2}(\.\d+)?)?(Z|[+-]\d{2}:\d{2}))?$/,
    "must be RFC3339 with an offset (2026-09-15T18:00:00+09:00) or a date (2026-09-15)",
  )
  .refine((s) => !Number.isNaN(Date.parse(s)), "must be a date that exists");

// httpsURL is a full https URL: the pattern for the JSON Schema, the refine for a URL that parses.
const httpsURL = z
  .string()
  .regex(/^https:\/\/[^\s/?#]+/, "must be a full https URL")
  .refine((s) => URL.canParse(s), "must be a full https URL");

const urls = z
  .array(httpsURL)
  .refine((a) => new Set(a).size === a.length, "must not repeat a URL")
  .meta({ uniqueItems: true });

// A section is its lines, one per item: no blank line, no heading, no "- " (a line is an item already), none
// over LINE_LIMIT. The limit
// counts characters, not UTF-16 units as maxLength in zod would, so it stays a refine and meta carries it
// into the JSON Schema.
const line = z
  .string()
  .regex(/\S/, "must not be blank")
  .regex(/^(?!#)/, "must not be a heading")
  .regex(/^(?!- )/, 'must not start with "- ": a line is one item already')
  .refine((s) => [...s].length <= LINE_LIMIT, `must be at most ${LINE_LIMIT} characters`)
  .meta({ maxLength: LINE_LIMIT });
const section = z.array(line);
const whyDropped = section.meta({
  description:
    "why the task was decided against; a dropped task needs one line at least, and no other task has one",
});

const nonBlank = z.string().regex(/^[^\n]*\S[^\n]*$/, "must be one non-blank line");

// Task is a task as the API returns it.
export const Task = z
  .strictObject({
    id: z
      .number()
      .int()
      .meta({ description: "the number the task keeps for life, unlike its slug" }),
    slug: kebab,
    title: nonBlank,
    status: Status,
    area: Area,
    project: kebab.nullable(),
    scheduled: scheduled.nullable(),
    waiting: nonBlank.nullable(),
    tickets: urls,
    prs: urls,
    links: urls,
    why_dropped: whyDropped,
    memo: section.meta({
      description: "what the owner did or thought while working on the task, one note per line",
    }),
    created: z.string(),
    updated: z.string(),
  })
  .meta({ id: "Task" });
export type Task = z.output<typeof Task>;

// TaskCreate is POST /api/tasks: a new task always starts as todo.
export const TaskCreate = z.strictObject({
  slug: kebab,
  title: nonBlank,
  area: Area,
  project: kebab.optional(),
  scheduled: scheduled.optional(),
  waiting: nonBlank.optional(),
  tickets: urls.default([]),
  prs: urls.default([]),
  links: urls.default([]),
});

// TaskPatch is PATCH /api/tasks/:id: any field, where null drops an optional value and a list replaces
// the one stored.
export const TaskPatch = Task.omit({ id: true, created: true, updated: true }).partial();

// Habit is a habit as GET /api/habits returns it: done on the day asked.
export const Habit = z
  .strictObject({ id: z.number().int(), title: nonBlank, done: z.boolean() })
  .meta({ id: "Habit" });
export type Habit = z.output<typeof Habit>;

// HabitCreate is POST /api/habits.
export const HabitCreate = Habit.pick({ title: true });

// HabitPatch is PATCH /api/habits/:id: a new title; its checks stay.
export const HabitPatch = Habit.pick({ title: true });

// day is a calendar day in Asia/Tokyo, the day a habit is checked on.
export const day = z
  .string()
  .regex(/^\d{4}-\d{2}-\d{2}$/, "must be a date (2026-10-26)")
  .refine((s) => !Number.isNaN(Date.parse(s)), "must be a date that exists");

// Item is one line of a task's checklist as /api/items returns it.
export const Item = z
  .strictObject({
    id: z.number().int(),
    task_id: z.number().int(),
    position: z.number().meta({ description: "the order in the checklist, smallest first" }),
    text: nonBlank,
    done: z.boolean(),
    created: z.string(),
    updated: z.string(),
  })
  .meta({ id: "Item" });
export type Item = z.output<typeof Item>;

// ItemCreate is POST /api/items: a new line, not done, last in its task's checklist.
export const ItemCreate = Item.pick({ task_id: true, text: true });

// ItemPatch is PATCH /api/items/:id: any field but the task it belongs to.
export const ItemPatch = Item.pick({ text: true, position: true, done: true }).partial();

// whole checks what no single field can: a dropped task says why, and only a dropped task does. It has no
// JSON Schema form, so whyDropped says it in words.
export function whole(t: Pick<Task, "status" | "why_dropped">): string | undefined {
  if (t.status === "dropped" && t.why_dropped.length === 0)
    return "a dropped task needs why_dropped";
  if (t.status !== "dropped" && t.why_dropped.length > 0)
    return `a ${t.status} task has no why_dropped`;
}
