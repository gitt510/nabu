import type { Item } from "../api";
import { fmtDay } from "../time";
import { Caption, Empty, Group, ListRow, Stack } from "./ui";

/** task の一覧。1 行は題と、待ちの印と、起票日。放っておかれた task が分かるように */
export function TaskList({ items, selected, onSelect }: { items: Item[]; selected: string; onSelect: (slug: string) => void }) {
  if (!items.length) return <Empty title="No tasks" description="Try clearing the search or filters." />;
  return (
    <Stack gap={2}>
      {items.map((t) => (
        <ListRow
          key={t.slug}
          id={`task-${t.slug}`}
          label={t.title}
          active={t.slug === selected}
          onClick={() => onSelect(t.slug)}
          aside={
            <Group gap={4} wrap="nowrap">
              {t.waiting ? "⏳" : null}
              <Caption mono>{fmtDay(t.created)}</Caption>
            </Group>
          }
        />
      ))}
    </Stack>
  );
}
