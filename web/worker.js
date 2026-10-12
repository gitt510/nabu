// Every path gets the one page; the viewer routes by hash. no-store, so a reload shows the latest publish.
export default {
  async fetch(request, env) {
    const page = await env.VIEW.get("index.html");
    if (!page) return new Response("not published yet: run `just publish`\n", { status: 404 });
    return new Response(page.body, {
      headers: { "content-type": "text/html; charset=utf-8", "cache-control": "no-store" },
    });
  },
};
