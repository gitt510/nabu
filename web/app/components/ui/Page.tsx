import type { ReactNode } from "react";
import { Box } from "@mantine/core";

const WIDTHS = { regular: "52rem", narrow: "36rem" } as const;

/** page の本文。読みやすい最大幅と、画面幅に合わせた余白を持つ。narrow は短い行が並ぶだけの page に使う */
export const Page = ({ width = "regular", children }: { width?: keyof typeof WIDTHS; children: ReactNode }) => (
  <Box maw={WIDTHS[width]} px={{ base: "md", sm: "xl" }} py="lg">
    {children}
  </Box>
);
