import type { Item } from "../api";
import { fmtDay } from "../time";
import { Caption, Checkbox, Empty, Group, ListRow, Stack } from "./ui";

// 印の読み上げ：今の状態から何をするか
const MARK_LABEL = { todo: "Mark done", done: "Reopen", dropped: "Dropped" } as const;

/** task の一覧。1 行は完了の印と題と、待ちの印と、起票日。放っておかれた task が分かるように。dropped の task は ✕ を出し、開き直しは詳細でする */
export function TaskList({ items, selected, onSelect, onCheck }: { items: Item[]; selected: string; onSelect: (id: string) => void; onCheck: (t: Item, done: boolean) => void }) {
  if (!items.length) return <Empty title="No tasks" description="Try clearing the search or filters." />;
  return (
    <Stack gap={2}>
      {items.map((t) => (
        <ListRow
          key={t.id}
          id={`task-${t.id}`}
          lead={<Checkbox bare label={MARK_LABEL[t.status]} checked={t.status === "done"} crossed={t.status === "dropped"} onChange={(v) => onCheck(t, v)} />}
          label={t.title}
          active={String(t.id) === selected}
          onClick={() => onSelect(String(t.id))}
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
