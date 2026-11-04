import type { ReactNode } from "react";
import { Flex, Group, Text, Title } from "@mantine/core";

// 題の 1 行の高さ。aside をこの高さの中で縦の中央に置き、1 行目と中央線を揃える
const LINE = 1.35;

/**
 * page と task の題。number は題のあとに薄く添える番号（task の #6 など）。aside は右端に置く操作（task を閉じるなど）で、
 * 題が折り返しても 1 行目に揃う。muted は閉じたものの題を薄く出す
 */
export const Heading = ({
  children,
  number,
  aside,
  muted,
}: {
  children: ReactNode;
  number?: string;
  aside?: ReactNode;
  muted?: boolean;
}) => {
  const title = (
    <Title order={1} size="h2" lh={LINE} c={muted ? "dimmed" : undefined}>
      {children}
      {number ? (
        <Text span inherit c="dimmed" fw={400}>
          {" "}
          {number}
        </Text>
      ) : null}
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
