import type { ReactNode } from "react";
import { Button as MantineButton } from "@mantine/core";

/** 操作の button。small は一覧の行の中に置く大きさ */
export const Button = ({
  children,
  type = "button",
  small = false,
  onClick,
}: {
  children: ReactNode;
  type?: "button" | "submit";
  small?: boolean;
  onClick?: () => void;
}) => (
  <MantineButton type={type} size={small ? "xs" : "sm"} onClick={onClick}>
    {children}
  </MantineButton>
);
