import type { ReactNode } from "react";
import { Stack, Text } from "@mantine/core";

/** 題のついた文章の 1 段。Section と違って枠を持たず、本文の流れの中に置く */
export const Passage = ({ title, children }: { title: string; children?: ReactNode }) => (
  <Stack gap={6}>
    <Text fw={700}>{title}</Text>
    {children}
  </Stack>
);
