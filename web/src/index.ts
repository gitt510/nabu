// The Worker behind Access: the viewer page comes from dist/ as static assets before this runs; this serves
// the API and, until the viewer reads the API, the data the publish workflow uploads to R2.
import { Hono } from "hono";
import { tasks } from "./tasks";

const app = new Hono<{ Bindings: Env }>()
  .route("/api/tasks", tasks)
  // no-store, so a reload shows the latest publish.
  .get("/tasks.json", async (c) => {
    const data = await c.env.VIEW.get("tasks.json");
    if (!data) return c.text("not published yet: push the notebook with the publish workflow in place\n", 404);
    return c.body(data.body, 200, { "content-type": "application/json; charset=utf-8", "cache-control": "no-store" });
  })
  .notFound((c) => c.text("not found\n", 404));

export default app;
