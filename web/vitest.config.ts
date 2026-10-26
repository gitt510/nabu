// Runs the tests inside workerd with the bindings of wrangler.jsonc, each test file on a fresh local D1
// that test/setup.ts migrates.
import { cloudflareTest, readD1Migrations } from "@cloudflare/vitest-plugin";
import { defineConfig } from "vitest/config";

export default defineConfig(async () => ({
  plugins: [
    cloudflareTest({
      wrangler: { configPath: "./wrangler.jsonc" },
      miniflare: {
        bindings: {
          TEST_MIGRATIONS: await readD1Migrations("migrations"),
          // Stand-ins for the vars just deploy passes; test/access.ts signs tokens for them.
          CF_ACCESS_TEAM_DOMAIN: "https://nabu-test.cloudflareaccess.com",
          CF_ACCESS_AUD: "nabu-test-aud",
        },
      },
    }),
  ],
  test: { setupFiles: ["./test/setup.ts"] },
}));
