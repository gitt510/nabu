// The Worker behind Access: the viewer page comes from dist/ as static assets before this runs; this serves
// the API and the data the page reads, both from D1.
import { Hono } from "hono";
import { access } from "./access";
import { tasks } from "./tasks";
import { view } from "./view";

const app = new Hono<{ Bindings: Env }>()
  .use(access)
  .route("/api/tasks", tasks)
  .route("/", view)
  .notFound((c) => c.text("not found\n", 404));

export default app;
