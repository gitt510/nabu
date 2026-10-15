// The page is a static asset; this serves only its data, which the publish workflow uploads to R2.
// no-store, so a reload shows the latest publish.
export default {
  async fetch(request, env) {
    if (new URL(request.url).pathname !== "/tasks.json") return new Response("not found\n", { status: 404 });
    const data = await env.VIEW.get("tasks.json");
    if (!data) return new Response("not published yet: push the notebook with the publish workflow in place\n", { status: 404 });
    return new Response(data.body, {
      headers: { "content-type": "application/json; charset=utf-8", "cache-control": "no-store" },
    });
  },
};
