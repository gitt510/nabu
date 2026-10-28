import type { ReactNode } from "react";
import { Group, Paper, Stack, Text } from "@mantine/core";

/** page の中の 1 区画。枠の中に、題と右端の補足、その下に中身を置く */
export const Section = ({ title, aside, children }: { title: string; aside?: ReactNode; children: ReactNode }) => (
  <Paper withBorder radius="md" p="md">
    <Stack gap="sm">
      <Group justify="space-between" wrap="nowrap">
        <Text fw={700}>{title}</Text>
        {aside}
      </Group>
      {children}
    </Stack>
  </Paper>
);
