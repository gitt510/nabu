// The task contract: what /api/tasks accepts and returns. Every rule a write must meet lives here, so a task
// out of shape cannot reach the database, and the page reads its fields from here as the Worker does.
import { z } from "zod";

export const STATUSES = ["inbox", "doing", "done", "dropped"] as const;
export const AREAS = ["work", "personal"] as const;

// LINE_LIMIT is the most characters a section line may carry: one fact a reader takes in at a glance.
export const LINE_LIMIT = 45;

// SECTIONS are SCQA, the introduction Barbara Minto lays out in The Pyramid Principle, then the reason a
// dropped task was decided against.
export const SECTIONS = ["situation", "complication", "question", "answer", "why_dropped"] as const;
export const LISTS = ["tickets", "prs", "links"] as const;

export type Status = (typeof STATUSES)[number];

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
const whyDropped = section.meta({ description: "why the task was decided against; a dropped task needs one line at least" });

const nonBlank = z.string().regex(/^[^\n]*\S[^\n]*$/, "must be one non-blank line");

// Task is a task as the API returns it.
export const Task = z.strictObject({
  slug: kebab,
  title: nonBlank,
  status: z.enum(STATUSES),
  area: z.enum(AREAS),
  project: kebab.nullable(),
  scheduled: scheduled.nullable(),
  waiting: nonBlank.nullable(),
  tickets: urls,
  prs: urls,
  links: urls,
  situation: section,
  complication: section,
  question: section,
  answer: section,
  why_dropped: whyDropped,
  created: z.string(),
  updated: z.string(),
});
export type Task = z.output<typeof Task>;

// TaskCreate is POST /api/tasks: a new task always starts in inbox.
export const TaskCreate = z.strictObject({
  slug: kebab,
  title: nonBlank,
  area: z.enum(AREAS),
  project: kebab.optional(),
  scheduled: scheduled.optional(),
  waiting: nonBlank.optional(),
  tickets: urls.default([]),
  prs: urls.default([]),
  links: urls.default([]),
  situation: section.default([]),
  complication: section.default([]),
  question: section.default([]),
  answer: section.default([]),
  why_dropped: whyDropped.default([]),
});

// TaskPatch is PATCH /api/tasks/:slug: any field, where null drops an optional value and a list replaces
// the one stored.
export const TaskPatch = Task.omit({ created: true, updated: true }).partial();

// Habit is a habit as GET /api/habits returns it: done on the day asked.
export const Habit = z.strictObject({ id: z.number().int(), title: nonBlank, done: z.boolean() });
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

// whole checks what no single field can: a dropped task says why. It has no JSON Schema form, so whyDropped
// says it in words.
export function whole(t: Pick<Task, "status" | "why_dropped">): string | undefined {
  if (t.status === "dropped" && t.why_dropped.length === 0) return "a dropped task needs why_dropped";
}
