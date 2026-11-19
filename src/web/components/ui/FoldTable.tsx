import type { ReactNode } from "react";
import { Box, Group, Table, Text, UnstyledButton } from "@mantine/core";
import { useMediaQuery } from "@mantine/hooks";

import { ChevronDownIcon, ChevronUpIcon } from "./icons";

export type FoldColumn = {
  id: string;
  header: string;
  /** 並べ替えの向き。onSort があれば見出しを押して切り替えられる */
  sorted?: false | "asc" | "desc";
  onSort?: (event: unknown) => void;
  /** 右に寄せる列（日付など） */
  end?: boolean;
  /** 狭い画面でも残す列。ほかの列は狭い画面では出さない */
  keep?: boolean;
};

export type FoldTableRow = {
  id: string;
  /** 行の要素の id。URL から開いた行まで scroll するときの目印 */
  anchor: string;
  cells: Record<string, ReactNode>;
  open: boolean;
  onToggle: () => void;
  /** 開いた行の右端に置く操作。行を押して閉じないよう、ここの click は行に伝えない */
  actions?: ReactNode;
  /** 開いたときに行の下に、全部の列にまたがって出す中身 */
  detail: ReactNode;
};

const SortMark = ({ sorted }: { sorted: FoldColumn["sorted"] }) =>
  sorted === "asc" ? (
    <ChevronUpIcon size={14} />
  ) : sorted === "desc" ? (
    <ChevronDownIcon size={14} />
  ) : null;

/**
 * 行をその場で開く表。見出しを押すと並べ替え、行を押すと行の下に中身を開く（もう一度押すと閉じる）。
 * 開いた行は背景を変えず、最初の列を太字にする。中身は一段下げ、左の線でどの行のものかを示す。
 * 幅が狭いと keep の列だけを残す。閉じた中身は描かない（開いていない行の中身は読み込まない）
 */
export function FoldTable({ columns, rows }: { columns: FoldColumn[]; rows: FoldTableRow[] }) {
  const narrow = useMediaQuery("(max-width: 48em)") ?? false;
  const shown = narrow ? columns.filter((c) => c.keep) : columns;
  return (
    <Table verticalSpacing="xs">
      <Table.Thead>
        <Table.Tr>
          {shown.map((c) => (
            <Table.Th key={c.id} ta={c.end ? "right" : undefined}>
              {c.onSort ? (
                <UnstyledButton onClick={c.onSort}>
                  <Group gap={4} wrap="nowrap" justify={c.end ? "flex-end" : undefined}>
                    <Text span size="xs" fw={700} c="dimmed">
                      {c.header}
                    </Text>
                    <SortMark sorted={c.sorted} />
                  </Group>
                </UnstyledButton>
              ) : (
                <Text span size="xs" fw={700} c="dimmed">
                  {c.header}
                </Text>
              )}
            </Table.Th>
          ))}
          <Table.Th w={1} aria-label="Actions" />
        </Table.Tr>
      </Table.Thead>
      <Table.Tbody>
        {rows.flatMap((r) => [
          <Table.Tr
            key={r.id}
            id={r.anchor}
            onClick={r.onToggle}
            aria-expanded={r.open}
            style={{ cursor: "pointer" }}
          >
            {shown.map((c, i) => (
              <Table.Td key={c.id} ta={c.end ? "right" : undefined}>
                {i === 0 ? (
                  <Text span inherit fw={r.open ? 700 : 400}>
                    {r.cells[c.id]}
                  </Text>
                ) : (
                  r.cells[c.id]
                )}
              </Table.Td>
            ))}
            {/* menu は portal に開くが、React の event はここを通って伝わるので、ここで止める */}
            <Table.Td py={0} onClick={(e) => e.stopPropagation()}>
              {r.open ? r.actions : null}
            </Table.Td>
          </Table.Tr>,
          r.open ? (
            <Table.Tr key={`${r.id}-detail`}>
              <Table.Td colSpan={shown.length + 1}>
                <Box
                  ml="lg"
                  my="xs"
                  pl="md"
                  pr="sm"
                  py="xs"
                  style={{ borderLeft: "2px solid var(--mantine-color-default-border)" }}
                >
                  {r.detail}
                </Box>
              </Table.Td>
            </Table.Tr>
          ) : null,
        ])}
      </Table.Tbody>
    </Table>
  );
}
