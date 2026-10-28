import type { ReactNode } from "react";
import { Anchor } from "@mantine/core";

/** 外へ出る link。別の tab で開く */
export const ExternalLink = ({ href, children }: { href: string; children: ReactNode }) => (
  <Anchor href={href} target="_blank" rel="noopener" size="sm">
    {children}
  </Anchor>
);
