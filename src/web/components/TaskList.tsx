import type { TaskView } from "../api";
import { fmtDay } from "../time";
import { Caption, EmptyState, Group, ListRow, Stack } from "./ui";

/**
 * task の一覧。1 行は題と、待ちの印と、起票日。放っておかれた task が分かるように。
 * updated のときは直した日を出す
 */
export function TaskList({
  items,
  selected,
  onSelect,
  updated,
}: {
  items: TaskView[];
  selected: string;
  onSelect: (id: string) => void;
  updated: boolean;
}) {
  if (!items.length) return <EmptyState title="No tasks" description="Try other filters." />;
  return (
    <Stack gap={2}>
      {items.map((t) => (
        <ListRow
          key={t.id}
          id={`task-${t.id}`}
          label={t.title}
          active={String(t.id) === selected}
          onClick={() => onSelect(String(t.id))}
          aside={
            <Group gap={4} wrap="nowrap">
              {t.waiting ? "⏳" : null}
              <Caption mono>{fmtDay(updated ? t.updated : t.created)}</Caption>
            </Group>
          }
        />
      ))}
    </Stack>
  );
}
