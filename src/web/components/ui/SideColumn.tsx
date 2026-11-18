import type { ReactNode } from "react";
import {
  Box,
  Collapse,
  Divider,
  Group,
  ScrollArea,
  Stack,
  Text,
  UnstyledButton,
} from "@mantine/core";
import { useDisclosure, useMediaQuery } from "@mantine/hooks";

import { ChevronDownIcon } from "./icons";

/**
 * 左の細い列と、右の本文の 2 区画。間に端から端までの区切りの線を引き、本文だけが scroll する。
 * 左には一覧を絞る Facet などを縦に積む。幅が狭いと左の列を本文の上に畳み、summary（今の絞り込み）だけを見せる。
 * 押すと開いて中身を横に並べ、もう一度押すと畳む
 */
export function SideColumn({
  side,
  summary,
  children,
}: {
  side: ReactNode;
  summary: string;
  children: ReactNode;
}) {
  const narrow = useMediaQuery("(max-width: 48em)") ?? false;
  const [opened, { toggle }] = useDisclosure(false);
  if (narrow)
    return (
      <Stack gap={0}>
        <UnstyledButton px="md" py="sm" onClick={toggle} aria-expanded={opened}>
          <Group justify="space-between" wrap="nowrap">
            <Text size="sm">{summary}</Text>
            <Box
              c="dimmed"
              style={{ transform: `rotate(${opened ? 180 : 0}deg)`, transition: "transform 150ms" }}
            >
              <ChevronDownIcon size={16} />
            </Box>
          </Group>
        </UnstyledButton>
        <Collapse expanded={opened}>
          <Group align="flex-start" grow px="md" pb="md">
            {side}
          </Group>
        </Collapse>
        <Divider />
        {children}
      </Stack>
    );
  return (
    <Group wrap="nowrap" align="stretch" gap={0} h="100%">
      <Stack w={220} flex="none" gap="lg" p="md">
        {side}
      </Stack>
      <Divider orientation="vertical" />
      <ScrollArea flex={1} miw={0}>
        <Box>{children}</Box>
      </ScrollArea>
    </Group>
  );
}
