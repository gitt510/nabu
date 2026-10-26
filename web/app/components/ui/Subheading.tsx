import type { ReactNode } from "react";
import { Title } from "@mantine/core";

/** 本文の中の小見出し */
export const Subheading = ({ children }: { children: ReactNode }) => (
  <Title order={2} size="h4" mt="lg" mb="xs">
    {children}
  </Title>
);
