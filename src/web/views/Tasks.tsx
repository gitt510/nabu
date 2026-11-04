import { useMemo, useState } from "react";

import { AREAS, STATUSES } from "../../api/schema";
import type { Item } from "../api";
import { STATUS_LABEL } from "../components/states";
import { TaskDetail } from "../components/TaskDetail";
import { TaskList } from "../components/TaskList";
import {
  Choice,
  Empty,
  ListDetail,
  Page,
  Stack,
  useHotkeys,
  useMediaQuery,
} from "../components/ui";
import { instant } from "../time";

/** 絞り込み。area の "all" は絞らない。最初は todo だけを見せる */
type Filters = { status: string; area: string };
const START: Filters = { status: "todo", area: "all" };

const hit = (t: Item, f: Filters) =>
  t.status === f.status && (f.area === "all" || t.area === f.area);

/**
 * task を並べる page。一覧の上に常に出した入力で status と area を絞る。todo だけなら作った日順、
 * 閉じた task を含むなら直した日順に、新しいものから並べる（閉じた日は持たないので、直した日をその代わりにする）
 */
export function Tasks({
  tasks,
  selected,
  onSelect,
  onChanged,
}: {
  tasks: Item[];
  selected: string;
  onSelect: (id: string, push?: boolean) => void;
  onChanged: () => void;
}) {
  const [filters, setFilters] = useState(START);
  const narrow = useMediaQuery("(max-width: 48em)") ?? false;
  const byUpdated = filters.status !== "todo";

  const items = useMemo(() => {
    const when = (t: Item) => instant(byUpdated ? t.updated : t.created).getTime();
    return tasks.filter((t) => hit(t, filters)).sort((a, b) => when(b) - when(a));
  }, [tasks, filters, byUpdated]);
  const task = tasks.find((t) => String(t.id) === selected);

  const open = (id: string) => {
    onSelect(id, narrow);
    document.getElementById(`task-${id}`)?.scrollIntoView({ block: "nearest" });
  };
  const step = (delta: number) => {
    const at = items.findIndex((t) => String(t.id) === selected);
    const next = items[Math.min(items.length - 1, Math.max(0, at + delta))];
    if (next) open(String(next.id));
  };

  useHotkeys([
    ["ArrowDown", () => step(1)],
    ["ArrowUp", () => step(-1)],
  ]);

  const set = (f: Partial<Filters>) => setFilters({ ...filters, ...f });

  const head = (
    <Stack gap="sm">
      <Choice
        label="Status"
        data={STATUSES.map((v) => ({ value: v, label: STATUS_LABEL[v] }))}
        value={filters.status}
        onChange={(status) => set({ status })}
      />
      <Choice
        label="Area"
        data={[{ value: "all", label: "All" }, ...AREAS.map((a) => ({ value: a, label: a }))]}
        value={filters.area}
        onChange={(area) => set({ area })}
      />
    </Stack>
  );

  return (
    <ListDetail
      head={head}
      list={<TaskList items={items} selected={selected} onSelect={open} updated={byUpdated} />}
      showDetail={!!task}
      detail={
        task ? (
          <TaskDetail
            key={task.id}
            task={task}
            onBack={narrow ? () => onSelect("") : undefined}
            onChanged={onChanged}
          />
        ) : (
          <Page>
            <Empty title="No task selected" description="Pick one from the list, or press ↓ / ↑." />
          </Page>
        )
      }
    />
  );
}
