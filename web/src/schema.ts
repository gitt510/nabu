// The task contract: what /api/tasks accepts and returns. Every rule a write must meet lives here,
// so a task out of shape cannot reach the database.
import { z } from "zod";

export const STATUSES = ["inbox", "doing", "done", "dropped"] as const;

// LINE_LIMIT is the most characters a section line may carry: one fact a reader takes in at a glance.
export const LINE_LIMIT = 30;

// SECTIONS are SCQA, the introduction Barbara Minto lays out in The Pyramid Principle, then the reason a
// dropped task was decided against.
export const SECTIONS = ["situation", "complication", "question", "answer", "why_dropped"] as const;
export const LISTS = ["tickets", "prs", "links"] as const;

const kebab = z.string().regex(/^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$/, "must be lowercase kebab-case");

// scheduled is the planned work time: RFC3339 with an offset, or a date for the whole day.
const scheduled = z
  .string()
  .refine(
    (s) =>
      (/^\d{4}-\d{2}-\d{2}$/.test(s) || /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(:\d{2}(\.\d+)?)?(Z|[+-]\d{2}:\d{2})$/.test(s)) &&
      !Number.isNaN(Date.parse(s)),
    "must be RFC3339 with an offset (2026-09-15T18:00:00+09:00) or a date (2026-09-15)",
  );

const httpsURL = z.string().refine((s) => {
  try {
    const u = new URL(s);
    return u.protocol === "https:" && u.host !== "";
  } catch {
    return false;
  }
}, "must be a full https URL");

const urls = z.array(httpsURL).refine((a) => new Set(a).size === a.length, "must not repeat a URL");

// A section is its markdown lines, one per item: no blank line, no heading, none over LINE_LIMIT.
const line = z
  .string()
  .refine((s) => s.trim() !== "", "must not be blank")
  .refine((s) => !s.startsWith("#"), "must not be a heading")
  .refine((s) => [...s].length <= LINE_LIMIT, `must be at most ${LINE_LIMIT} characters`);
const section = z.array(line);

const nonBlank = z.string().refine((s) => s.trim() !== "" && !s.includes("\n"), "must be one non-blank line");

// TaskCreate is POST /api/tasks: a new task always starts in inbox.
export const TaskCreate = z.strictObject({
  slug: kebab,
  title: nonBlank,
  area: z.enum(["work", "personal"]),
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
  why_dropped: section.default([]),
});

// TaskPatch is PATCH /api/tasks/:slug: any field, where null drops an optional value and a list replaces
// the one stored.
export const TaskPatch = z.strictObject({
  slug: kebab.optional(),
  title: nonBlank.optional(),
  status: z.enum(STATUSES).optional(),
  area: z.enum(["work", "personal"]).optional(),
  project: kebab.nullable().optional(),
  scheduled: scheduled.nullable().optional(),
  waiting: nonBlank.nullable().optional(),
  tickets: urls.optional(),
  prs: urls.optional(),
  links: urls.optional(),
  situation: section.optional(),
  complication: section.optional(),
  question: section.optional(),
  answer: section.optional(),
  why_dropped: section.optional(),
});

export const TaskQuery = z.strictObject({
  status: z.enum(STATUSES).optional(),
  q: z.string().optional(),
});

export type Status = (typeof STATUSES)[number];

// Task is a task as the API returns it.
export type Task = {
  slug: string;
  title: string;
  status: Status;
  area: "work" | "personal";
  project: string | null;
  scheduled: string | null;
  waiting: string | null;
  tickets: string[];
  prs: string[];
  links: string[];
  situation: string[];
  complication: string[];
  question: string[];
  answer: string[];
  why_dropped: string[];
  created: string;
  updated: string;
};

// whole checks what no single field can: a dropped task says why.
export function whole(t: Pick<Task, "status" | "why_dropped">): string | undefined {
  if (t.status === "dropped" && t.why_dropped.length === 0) return "a dropped task needs why_dropped";
}
