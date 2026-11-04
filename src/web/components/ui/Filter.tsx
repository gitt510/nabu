import type { ReactNode } from "react";
import { Button, Group, Pill, Popover, Stack } from "@mantine/core";

import { FilterIcon } from "./icons";

/** かけている絞り込み 1 つ：見出しと、外す操作 */
export type FilterChip = { label: string; onClear: () => void };

/**
 * 一覧の絞り込み。Filter を押すと children の小窓が開き、かけている絞り込みは
 * ボタンの横に × で外せる印として並ぶ。ボタンは内側の余白のぶん左へずらし、icon の左端を周りの文字に揃える
 */
export const Filter = ({ chips, children }: { chips: FilterChip[]; children: ReactNode }) => (
  <Group gap={6}>
    <Popover position="bottom-start" width={380} shadow="md">
      <Popover.Target>
        <Button
          size="xs"
          ml="calc(var(--button-padding-x) / -1.5)"
          variant="subtle"
          color="gray"
          leftSection={<FilterIcon size={14} />}
        >
          Filter
        </Button>
      </Popover.Target>
      <Popover.Dropdown>
        <Stack gap="md">{children}</Stack>
      </Popover.Dropdown>
    </Popover>
    {chips.map((c) => (
      <Pill
        key={c.label}
        withRemoveButton
        onRemove={c.onClear}
        removeButtonProps={{ "aria-label": `Clear ${c.label}` }}
      >
        {c.label}
      </Pill>
    ))}
  </Group>
);
