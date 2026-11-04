import type { ReactNode } from "react";
import { Divider, ScrollArea, Stack } from "@mantine/core";

import { Page } from "./Page";

/**
 * 上の head と下の本文に分けた 1 枚。head は動かず、本文だけが scroll する。
 * 区切りの線は余白の外に、端から端まで引く（一覧の pane の head と一覧の区切りと揃える）
 */
export const Pane = ({ head, children }: { head: ReactNode; children: ReactNode }) => (
  <Stack gap={0} h="100%">
    <Page>{head}</Page>
    <Divider />
    <ScrollArea flex={1}>
      <Page>{children}</Page>
    </ScrollArea>
  </Stack>
);
