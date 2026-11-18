import type { ReactNode } from "react";
import { Text, UnstyledButton } from "@mantine/core";

/** 押せる文字。見た目は本文のままで、空いている幅いっぱいに伸びる。行の文字を押してその場で直すときに使う */
export const TextButton = ({
  label,
  children,
  onClick,
}: {
  label: string;
  children: ReactNode;
  onClick: () => void;
}) => (
  <UnstyledButton flex={1} miw={0} aria-label={label} onClick={onClick}>
    <Text span>{children}</Text>
  </UnstyledButton>
);
