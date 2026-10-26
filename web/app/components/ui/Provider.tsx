import type { ReactNode } from "react";
import { Badge, createTheme, Kbd, localStorageColorSchemeManager, MantineProvider } from "@mantine/core";

// 書体は Noto Sans JP / Mono（main.tsx が package から読む）。色は Mantine の既定の palette だけ。
// 文字は 14px を床にする：xs を sm と同じ 14px に上げ、どの部品が xs を使っても 14px より小さくならない。
// Badge と Kbd は段階表を使わず自前の大きさ（10px 前後）を持つので、ここで床に合わせる
const theme = createTheme({
  primaryColor: "teal",
  fontSizes: { xs: "0.875rem", sm: "0.875rem", md: "1rem", lg: "1.125rem", xl: "1.25rem" },
  fontFamily: '"Noto Sans JP Variable", -apple-system, BlinkMacSystemFont, sans-serif',
  fontFamilyMonospace: '"Noto Sans Mono Variable", ui-monospace, Menlo, monospace',
  components: {
    Badge: Badge.extend({ vars: () => ({ root: { "--badge-fz": "var(--mantine-font-size-xs)", "--badge-height": "1.375rem" } }) }),
    Kbd: Kbd.extend({ vars: () => ({ root: { "--kbd-fz": "var(--mantine-font-size-xs)" } }) }),
  },
  headings: { fontFamily: '"Noto Sans JP Variable", -apple-system, BlinkMacSystemFont, sans-serif' },
});

/** light / dark の明示選択だけを保存する key。index.html の inline script と共有する */
const colorSchemeManager = localStorageColorSchemeManager({ key: "nabu-color-scheme" });

export function Provider({ children }: { children: ReactNode }) {
  return (
    <MantineProvider theme={theme} colorSchemeManager={colorSchemeManager} defaultColorScheme="auto">
      {children}
    </MantineProvider>
  );
}
