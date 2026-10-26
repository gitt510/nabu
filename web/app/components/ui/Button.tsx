import type { ReactNode } from "react";
import { Button as MantineButton } from "@mantine/core";

const TONES = { primary: { variant: "filled" }, danger: { variant: "filled", color: "red" }, quiet: { variant: "default" } } as const;

/** 操作の button。tone は danger（消すなど戻せない操作）と quiet（やめる）。small は一覧の行の中に置く大きさ */
export const Button = ({
  children,
  type = "button",
  tone = "primary",
  small = false,
  onClick,
}: {
  children: ReactNode;
  type?: "button" | "submit";
  tone?: keyof typeof TONES;
  small?: boolean;
  onClick?: () => void;
}) => (
  <MantineButton type={type} size={small ? "xs" : "sm"} {...TONES[tone]} onClick={onClick}>
    {children}
  </MantineButton>
);
