// Runs the tests inside workerd with the bindings of wrangler.jsonc, each test file on a fresh local D1
// that src/api/test/setup.ts migrates.
import { cloudflareTest, readD1Migrations } from "@cloudflare/vitest-plugin";
import { defineConfig } from "vitest/config";

export default defineConfig(async () => ({
  plugins: [
    cloudflareTest({
      wrangler: { configPath: "./wrangler.jsonc" },
      miniflare: { bindings: { TEST_MIGRATIONS: await readD1Migrations("migrations") } },
    }),
  ],
  test: { setupFiles: ["./src/api/test/setup.ts"] },
}));
