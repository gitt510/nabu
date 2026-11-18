import type { ReactNode } from "react";
import { Box } from "@mantine/core";

/** checkbox の行の文字の列まで下げる。checklist の末尾に足す欄を、項目の文字と揃える（checkbox の幅と、文字までの余白） */
export const Indent = ({ children }: { children: ReactNode }) => (
  <Box pl="calc(1.25rem + var(--mantine-spacing-sm))">{children}</Box>
);
