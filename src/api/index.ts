// The Worker behind Access: the viewer page comes from dist/ as static assets before this runs; this serves
// the API, from D1, to the CLI and the page alike.
import { OpenAPIHono } from "@hono/zod-openapi";
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

const app = new OpenAPIHono<{ Bindings: Env }>();
app.use(access);
app.get("/api/schema", (c) => c.json(contract));
app.route("/api/tasks", tasks);
app.route("/api/habits", habits);
// /api/openapi.json is every route of the API as OpenAPI, built from the routes and their zod schemas.
app.doc31("/api/openapi.json", { openapi: "3.1.0", info: { title: "nabu", version: "1" } });
app.notFound((c) => c.text("not found\n", 404));

export default app;
