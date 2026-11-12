// Builds the viewer from src/web/ into dist/, which the Worker serves as static assets. In dev the page runs
// from the working tree and /api/* goes to the Worker that just dev runs from the working tree on the local
// D1. No Access stands in front of it, so the proxy puts the token cloudflared holds where Access would, and
// the Worker checks it as it checks every request in production: there is no way around the check.
import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { homedir } from "node:os";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

const api = () => {
  const config = `${process.env.XDG_CONFIG_HOME || `${homedir()}/.config`}/nabu/config.json`;
  const { base_url: app } = JSON.parse(readFileSync(config, "utf8"));
  const token = () =>
    execFileSync("cloudflared", ["access", "token", `-app=${app}`], { encoding: "utf8" }).trim();
  return {
    target: "http://localhost:8788",
    changeOrigin: true,
    configure: (proxy: {
      on: (e: "proxyReq", f: (req: { setHeader: (k: string, v: string) => void }) => void) => void;
    }) => proxy.on("proxyReq", (req) => req.setHeader("cf-access-jwt-assertion", token())),
  };
};

export default defineConfig(({ command }) => ({
  root: "src/web",
  plugins: [react()],
  build: { outDir: "../../dist", emptyOutDir: true },
  server: { port: 8787, proxy: command === "serve" ? { "/api/": api() } : undefined },
}));
