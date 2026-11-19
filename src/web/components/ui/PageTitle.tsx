import type { ReactNode } from "react";
import { Group, Title } from "@mantine/core";

/** page の題。aside は右端に置く、その page の主な操作（足すなど） */
export const PageTitle = ({ children, aside }: { children: ReactNode; aside?: ReactNode }) => (
  <Group justify="space-between" wrap="nowrap">
    <Title order={1} size="h2" lh={1.35}>
      {children}
    </Title>
    {aside}
  </Group>
);
