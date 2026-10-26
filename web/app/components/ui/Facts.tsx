import type { ReactNode } from "react";
import { Box, Group } from "@mantine/core";

import { Caption } from "./Caption";

/** 名前と値の組を横に流す（task の properties） */
export const Facts = ({ items }: { items: { label: string; value: ReactNode }[] }) => (
  <Group gap="sm" component="dl" my={0}>
    {items.map((i) => (
      <Group key={i.label} gap={6} wrap="nowrap">
        <Box component="dt">
          <Caption mono>{i.label}</Caption>
        </Box>
        <Group component="dd" gap={4} ms={0}>
          {i.value}
        </Group>
      </Group>
    ))}
  </Group>
);
