// Builds the viewer from src/web/ into dist/, which the Worker serves as static assets. In dev the page runs
// from the working tree and /api/* goes to the deployed Worker the nabu CLI talks to, signed in with the
// Access token cloudflared holds, as the CLI is; writes from the dev page reach the production D1.
import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { homedir } from "node:os";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

const api = () => {
  const config = `${process.env.XDG_CONFIG_HOME || `${homedir()}/.config`}/nabu/config.json`;
  const { base_url: target } = JSON.parse(readFileSync(config, "utf8"));
  const token = () => execFileSync("cloudflared", ["access", "token", `-app=${target}`], { encoding: "utf8" }).trim();
  return {
    target,
    changeOrigin: true,
    configure: (proxy: { on: (e: "proxyReq", f: (req: { setHeader: (k: string, v: string) => void }) => void) => void }) =>
      proxy.on("proxyReq", (req) => req.setHeader("cf-access-token", token())),
  };
};

export default defineConfig(({ command }) => ({
  root: "src/web",
  plugins: [react()],
  build: { outDir: "../../dist", emptyOutDir: true },
  server: { port: 8787, proxy: command === "serve" ? { "/api/": api() } : undefined },
}));
