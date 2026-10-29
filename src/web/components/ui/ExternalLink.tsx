import type { ReactNode } from "react";
import { Anchor, Group } from "@mantine/core";

import type { IconComponent } from "./icons";

/** 外へ出る link。別の tab で開く。icon を渡すと、何への link かを文字の前に示す */
export const ExternalLink = ({ href, icon: Icon, children }: { href: string; icon?: IconComponent; children: ReactNode }) => (
  <Anchor href={href} target="_blank" rel="noopener" size="sm" display={Icon ? "inline-flex" : undefined} lh={Icon ? 1 : undefined}>
    {Icon ? (
      <Group component="span" gap={4} wrap="nowrap">
        <Icon size={14} />
        {children}
      </Group>
    ) : (
      children
    )}
  </Anchor>
);
