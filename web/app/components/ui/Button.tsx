import type { ReactNode } from "react";
import { Button as MantineButton } from "@mantine/core";

/** 操作の button */
export const Button = ({ children, type = "button", onClick }: { children: ReactNode; type?: "button" | "submit"; onClick?: () => void }) => (
  <MantineButton type={type} onClick={onClick}>
    {children}
  </MantineButton>
);
