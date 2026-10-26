import type { ReactNode } from "react";
import { Box } from "@mantine/core";

/** page の本文。読みやすい最大幅と、画面幅に合わせた余白を持つ */
export const Page = ({ children }: { children: ReactNode }) => (
  <Box maw="52rem" px={{ base: "md", sm: "xl" }} py="lg">
    {children}
  </Box>
);
