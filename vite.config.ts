// Builds the viewer from src/web/ into dist/, which the Worker serves as static assets. In dev the page runs
// from the working tree and /api/* goes to the Worker that just dev runs from the working tree on the local
// D1.
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

export default defineConfig(({ command }) => ({
  root: "src/web",
  plugins: [react()],
  build: { outDir: "../../dist", emptyOutDir: true },
  server: {
    port: 8787,
    proxy: command === "serve" ? { "/api/": "http://localhost:8788" } : undefined,
  },
}));
