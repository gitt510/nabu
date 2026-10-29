import type { ReactNode } from "react";
import { Flex, Group, Title } from "@mantine/core";

// 題の 1 行の高さ。aside をこの高さの中で縦の中央に置き、1 行目と中央線を揃える
const LINE = 1.35;

/** page と task の題。aside は右端に置く操作（task の status など）で、題が折り返しても 1 行目に揃う */
export const Heading = ({ children, aside }: { children: ReactNode; aside?: ReactNode }) => {
  const title = (
    <Title order={1} size="h2" lh={LINE}>
      {children}
    </Title>
  );
  if (!aside) return title;
  return (
    <Group justify="space-between" align="flex-start" wrap="nowrap" preventGrowOverflow={false}>
      {title}
      <Flex h={`calc(var(--mantine-h2-font-size) * ${LINE})`} align="center" flex="none">
        {aside}
      </Flex>
    </Group>
  );
};
