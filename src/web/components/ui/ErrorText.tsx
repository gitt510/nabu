import type { ReactNode } from "react";
import { Text } from "@mantine/core";

/** 書き込みを断られた理由 */
export const ErrorText = ({ children }: { children: ReactNode }) =>
  children ? (
    <Text size="sm" c="red" role="alert">
      {children}
    </Text>
  ) : null;
