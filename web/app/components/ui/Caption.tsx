import type { ReactNode } from "react";
import { Text } from "@mantine/core";

/** 補足。日付、件数、空の説明 */
export const Caption = ({ children, mono }: { children: ReactNode; mono?: boolean }) => (
  <Text span size="xs" c="dimmed" ff={mono ? "monospace" : undefined}>
    {children}
  </Text>
);
