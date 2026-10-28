import type { ReactNode } from "react";
import { Title } from "@mantine/core";

/** page と task の題 */
export const Heading = ({ children }: { children: ReactNode }) => (
  <Title order={1} size="h2" lh={1.35}>
    {children}
  </Title>
);
