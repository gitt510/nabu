// Formats the TypeScript in src/ and the config files at the root with oxfmt's defaults: the Prettier style,
// 100 columns. `pnpm fmt` writes, and `pnpm check` fails on a file it would change.
import { defineConfig } from "oxfmt";

export default defineConfig({});
