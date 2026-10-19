// The Worker behind Access: the viewer page comes from dist/ as static assets before this runs; this serves
// the API, from D1, to the CLI and the page alike.
import { Hono } from "hono";
import { access } from "./access";
import { tasks } from "./tasks";

const app = new Hono<{ Bindings: Env }>()
  .use(access)
  .route("/api/tasks", tasks)
  .notFound((c) => c.text("not found\n", 404));

export default app;
