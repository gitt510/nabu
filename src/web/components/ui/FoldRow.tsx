import type { ReactNode } from "react";
import { Box, Collapse, NavLink, Text } from "@mantine/core";

/**
 * 一覧の 1 行で、その場で開くもの。題と右端の補足の行を押すと、行の下に中身を広げる。
 * 中身は行の文字から一段下げ、左の線でどの行のものかを示す。開いた行は背景を変えず題を太字にし、線はグレーにする。閉じた中身は消す（開いていない行の中身は読み込まない）
 */
export const FoldRow = ({
  id,
  label,
  aside,
  open,
  onToggle,
  children,
}: {
  id: string;
  label: ReactNode;
  aside: ReactNode;
  open: boolean;
  onToggle: () => void;
  children: ReactNode;
}) => (
  <Box id={id}>
    <NavLink
      label={
        <Text span inherit fw={open ? 700 : 400}>
          {label}
        </Text>
      }
      rightSection={aside}
      onClick={onToggle}
      aria-expanded={open}
      noWrap
    />
    <Collapse expanded={open} keepMounted={false}>
      <Box
        ml="lg"
        mt="xs"
        mb="lg"
        pl="md"
        pr="sm"
        py="xs"
        style={{ borderLeft: "2px solid var(--mantine-color-default-border)" }}
      >
        {children}
      </Box>
    </Collapse>
  </Box>
);
