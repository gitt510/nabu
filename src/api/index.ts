// The Worker behind Access: the viewer page comes from dist/ as static assets before this runs; this serves
// the API, from D1, to the CLI and the page alike.
import { Hono } from "hono";
import { z } from "zod";
import { access } from "./access";
import { habits } from "./habits";
import { TaskCreate, TaskPatch } from "./schema";
import { tasks } from "./tasks";

// contract is the task contract as JSON Schema: create for task new, patch for the rest.
const contract = {
  create: z.toJSONSchema(TaskCreate, { io: "input", unrepresentable: "any" }),
  patch: z.toJSONSchema(TaskPatch, { io: "input", unrepresentable: "any" }),
};

const app = new Hono<{ Bindings: Env }>()
  .use(access)
  .get("/api/schema", (c) => c.json(contract))
  .route("/api/tasks", tasks)
  .route("/api/habits", habits)
  .notFound((c) => c.text("not found\n", 404));

export default app;
