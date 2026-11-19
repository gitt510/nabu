import type { ReactNode } from "react";
import { Box, Collapse, Group, NavLink, Text } from "@mantine/core";

/**
 * 一覧の 1 行で、その場で開くもの。題と右端の補足の行を押すと、行の下に中身を広げる。

 * actions は開いた行の右端、行の button の外に置く（button の中に button を入れない）。
 * 中身は行の文字から一段下げ、左の線でどの行のものかを示す。開いた行は背景を変えず題を太字にし、線はグレーにする。閉じた中身は消す（開いていない行の中身は読み込まない）
 */
export const FoldRow = ({
  id,
  label,
  aside,
  actions,
  open,
  onToggle,
  children,
}: {
  id: string;
  label: ReactNode;
  aside: ReactNode;
  actions?: ReactNode;
  open: boolean;
  onToggle: () => void;
  children: ReactNode;
}) => (
  <Box id={id}>
    <Group gap={4} wrap="nowrap">
      <NavLink
        flex={1}
        miw={0}
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
      {open ? actions : null}
    </Group>
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
