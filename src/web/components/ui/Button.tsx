import type { ReactNode } from "react";
import { Button as MantineButton } from "@mantine/core";

import type { IconComponent } from "./icons";

/** 操作の button。small は一覧の行の中に置く大きさ。icon は文字の前に添える */
export const Button = ({
  children,
  type = "button",
  small = false,
  icon: Icon,
  onClick,
}: {
  children: ReactNode;
  type?: "button" | "submit";
  small?: boolean;
  icon?: IconComponent;
  onClick?: () => void;
}) => (
  <MantineButton
    type={type}
    size={small ? "xs" : "sm"}
    leftSection={Icon ? <Icon size={16} /> : undefined}
    onClick={onClick}
  >
    {children}
  </MantineButton>
);
