import type { ReactNode } from "react";
import { Badge } from "@mantine/core";

/** 値の札。area、project、期限 */
export const Tag = ({ children, tone = "default" }: { children: ReactNode; tone?: "default" | "late" | "muted" }) => (
  <Badge size="sm" radius="sm" variant="light" color={tone === "late" ? "red" : tone === "muted" ? "gray" : undefined} tt="none">
    {children}
  </Badge>
);
