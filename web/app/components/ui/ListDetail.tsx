import type { ReactNode } from "react";
import { Box, Divider, Group, ScrollArea, Stack } from "@mantine/core";
import { useMediaQuery } from "@mantine/hooks";

/**
 * 一覧と詳細の 2 pane。それぞれが自分の中で scroll する。
 * 幅が狭いと 1 枚ずつになり、detail が開いているあいだは detail だけを出す
 */
export function ListDetail({ head, list, detail, showDetail }: { head: ReactNode; list: ReactNode; detail: ReactNode; showDetail: boolean }) {
  const narrow = useMediaQuery("(max-width: 48em)") ?? false;
  const listPane = (
    <Stack gap={0} h="100%" w={narrow ? "100%" : 340} miw={narrow ? undefined : 340}>
      <Stack gap="xs" p="sm">
        {head}
      </Stack>
      <ScrollArea flex={1} px="xs" pb="xs">
        {list}
      </ScrollArea>
    </Stack>
  );
  const detailPane = (
    <ScrollArea h="100%" flex={1} miw={0}>
      <Box maw="52rem" px={{ base: "md", sm: "xl" }} py="lg">
        {detail}
      </Box>
    </ScrollArea>
  );
  if (narrow) return showDetail ? detailPane : listPane;
  return (
    <Group wrap="nowrap" align="stretch" gap={0} h="100%">
      {listPane}
      <Divider orientation="vertical" />
      {detailPane}
    </Group>
  );
}
