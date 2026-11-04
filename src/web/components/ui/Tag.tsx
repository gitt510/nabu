import type { ReactNode } from "react";
import { Badge } from "@mantine/core";

/** 分類の札。area、project。状態は持たない */
export const Tag = ({ children }: { children: ReactNode }) => (
  <Badge size="sm" radius="sm" variant="default" tt="none">
    {children}
  </Badge>
);
