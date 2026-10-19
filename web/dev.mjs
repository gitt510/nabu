// Serves the viewer from the working tree at http://localhost:8787, rebuilt on every page load, and relays
// /api/* to the Worker the nabu CLI talks to, signed in with the Access token cloudflared holds, as the CLI is.
// The Worker and Access stay as deployed; writes from this page reach the production D1.
import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { createServer } from "node:http";
import { homedir } from "node:os";

const { url } = JSON.parse(readFileSync(`${homedir()}/.config/nabu/config.json`, "utf8"));
const dist = new URL("dist/", import.meta.url);
const types = { ".html": "text/html; charset=utf-8", ".js": "text/javascript; charset=utf-8" };

createServer(async (req, res) => {
  try {
    if (req.url.startsWith("/api/")) {
      const token = execFileSync("cloudflared", ["access", "token", `-app=${url}`], { encoding: "utf8" }).trim();
      const body = ["GET", "HEAD"].includes(req.method) ? undefined : Buffer.concat(await req.toArray());
      const up = await fetch(new URL(req.url, url), {
        method: req.method,
        headers: { "cf-access-token": token, "content-type": req.headers["content-type"] ?? "application/json" },
        body,
      });
      res.writeHead(up.status, { "content-type": up.headers.get("content-type") ?? "" });
      return res.end(Buffer.from(await up.arrayBuffer()));
    }
    const path = new URL(req.url, "http://localhost").pathname;
    if (path === "/") execFileSync(process.execPath, ["build.mjs"], { cwd: new URL(".", import.meta.url), stdio: "inherit" });
    const file = path === "/" ? "index.html" : path.slice(1);
    res.writeHead(200, { "content-type": types[file.slice(file.lastIndexOf("."))] ?? "application/octet-stream" });
    const data = readFileSync(new URL(file, dist));
    // The tab says dev, so it is not mistaken for the deployed viewer.
    res.end(file === "index.html" ? data.toString().replace("<title>nabu</title>", "<title>nabu (dev)</title>") : data);
  } catch (e) {
    res.writeHead(e.code === "ENOENT" ? 404 : 502, { "content-type": "text/plain; charset=utf-8" });
    res.end(e.code === "ENOENT" ? "not found" : `${e.message}\nsigned in? run cloudflared access login ${url}`);
  }
}).listen(8787, () => console.log(`http://localhost:8787 → ${url}`));
