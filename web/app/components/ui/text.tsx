import type { ReactNode } from "react";
import { Anchor, Badge, Text, Title } from "@mantine/core";

/** page と task の題 */
export const Heading = ({ children }: { children: ReactNode }) => (
  <Title order={1} size="h2" lh={1.35}>
    {children}
  </Title>
);

/** 本文の中の小見出し */
export const Subheading = ({ children }: { children: ReactNode }) => (
  <Title order={2} size="h4" mt="lg" mb="xs">
    {children}
  </Title>
);

/** 補足。日付、件数、空の説明 */
export const Caption = ({ children, mono }: { children: ReactNode; mono?: boolean }) => (
  <Text span size="xs" c="dimmed" ff={mono ? "monospace" : undefined}>
    {children}
  </Text>
);

/** 書き込みを断られた理由 */
export const ErrorText = ({ children }: { children: ReactNode }) =>
  children ? (
    <Text size="sm" c="red" role="alert">
      {children}
    </Text>
  ) : null;

/** 値の札。area、project、期限 */
export const Tag = ({ children, tone = "default" }: { children: ReactNode; tone?: "default" | "late" | "muted" }) => (
  <Badge size="sm" radius="sm" variant="light" color={tone === "late" ? "red" : tone === "muted" ? "gray" : undefined} tt="none">
    {children}
  </Badge>
);

/** 外へ出る link。別の tab で開く */
export const ExternalLink = ({ href, children }: { href: string; children: ReactNode }) => (
  <Anchor href={href} target="_blank" rel="noopener" size="sm">
    {children}
  </Anchor>
);
