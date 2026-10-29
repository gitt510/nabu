import type { ReactNode } from "react";
import { SimpleGrid } from "@mantine/core";

/** 同じ幅の 2 列。左が一覧、右がそこで選んだものの詳細。狭い画面で 1 列にするかは使う側が決める */
export const Columns = ({ children }: { children: [ReactNode, ReactNode] }) => (
  <SimpleGrid cols={2} spacing="lg">
    {children}
  </SimpleGrid>
);
