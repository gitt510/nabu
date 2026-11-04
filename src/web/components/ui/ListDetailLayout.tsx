import type { ReactNode } from "react";
import { Box, Divider, Group, ScrollArea, Stack } from "@mantine/core";
import { useMediaQuery } from "@mantine/hooks";

/**
 * 一覧と詳細の 2 pane。左は上の head（絞り込みなど）と下の一覧に分け、一覧と詳細がそれぞれ自分の中で scroll する（詳細の scroll は detail が持つ）。
 * 幅が狭いと 1 枚ずつになり、detail が開いているあいだは detail だけを出す
 */
export function ListDetailLayout({
  head,
  list,
  detail,
  showDetail,
}: {
  head: ReactNode;
  list: ReactNode;
  detail: ReactNode;
  showDetail: boolean;
}) {
  const narrow = useMediaQuery("(max-width: 48em)") ?? false;
  const listPane = (
    <Stack gap={0} h="100%" w={narrow ? "100%" : 340} miw={narrow ? undefined : 340}>
      {/* 左右は一覧の行の文字に揃える：一覧の余白（xs）と行の内側の余白（sm）の和 */}
      <Stack gap="xs" py="sm" px="calc(var(--mantine-spacing-xs) + var(--mantine-spacing-sm))">
        {head}
      </Stack>
      <Divider />
      <ScrollArea flex={1} p="xs">
        {list}
      </ScrollArea>
    </Stack>
  );
  // 詳細の余白と scroll は detail が自分で持つ（Pane や Page で）
  const detailPane = (
    <Box h="100%" flex={1} miw={0}>
      {detail}
    </Box>
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
