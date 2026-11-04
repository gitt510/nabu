import type { ReactNode } from "react";
import { Group, Text } from "@mantine/core";

import type { IconComponent } from "./icons";

/** 補足。日付、件数、空の説明。icon を渡すと、何の補足かを文字の前に示す */
export const Caption = ({
  children,
  mono,
  icon: Icon,
}: {
  children: ReactNode;
  mono?: boolean;
  icon?: IconComponent;
}) => {
  const text = (
    <Text span size="xs" c="dimmed" ff={mono ? "monospace" : undefined}>
      {children}
    </Text>
  );
  return Icon ? (
    <Group component="span" gap={4} wrap="nowrap" c="dimmed">
      <Icon size={14} />
      {text}
    </Group>
  ) : (
    text
  );
};
