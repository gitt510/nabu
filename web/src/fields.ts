// The task's fields as plain values and types, without zod, so the page can import them as well as the
// Worker. schema.ts holds the rules a write must meet.
export const STATUSES = ["inbox", "doing", "done", "dropped"] as const;
export const AREAS = ["work", "personal"] as const;

// LINE_LIMIT is the most characters a section line may carry: one fact a reader takes in at a glance.
export const LINE_LIMIT = 45;

// SECTIONS are SCQA, the introduction Barbara Minto lays out in The Pyramid Principle, then the reason a
// dropped task was decided against.
export const SECTIONS = ["situation", "complication", "question", "answer", "why_dropped"] as const;
export const LISTS = ["tickets", "prs", "links"] as const;

export type Status = (typeof STATUSES)[number];

// Task is a task as the API returns it.
export type Task = {
  slug: string;
  title: string;
  status: Status;
  area: (typeof AREAS)[number];
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
