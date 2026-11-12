import type { ReactNode } from "react";
import { Group, Stack, Text } from "@mantine/core";

/** 題のついた文章の 1 段。Section と違って枠を持たず、本文の流れの中に置く。aside は題の右端に添える */
export const Passage = ({
  title,
  aside,
  children,
}: {
  title: string;
  aside?: ReactNode;
  children?: ReactNode;
}) => (
  <Stack gap={6}>
    <Group justify="space-between" wrap="nowrap">
      <Text fw={700}>{title}</Text>
      {aside}
    </Group>
    {children}
  </Stack>
);
