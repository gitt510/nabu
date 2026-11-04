import { useMemo, useState } from "react";

import { AREAS, STATUSES } from "../../api/schema";
import type { Item } from "../api";
import { STATUS_LABEL } from "../components/states";
import { TaskDetail } from "../components/TaskDetail";
import { TaskList } from "../components/TaskList";
import {
  Choice,
  Empty,
  Filter,
  type FilterChip,
  ListDetail,
  Select,
  Switch,
  useHotkeys,
  useMediaQuery,
} from "../components/ui";
import { instant } from "../time";

/** 絞り込み。"all" と null と false は絞らない。最初は todo だけを見せる */
type Filters = { status: string; area: string; project: string | null; scheduled: boolean };
const START: Filters = { status: "todo", area: "all", project: null, scheduled: false };

const hit = (t: Item, f: Filters) =>
  (f.status === "all" || t.status === f.status) &&
  (f.area === "all" || t.area === f.area) &&
  (!f.project || t.project === f.project) &&
  (!f.scheduled || !!t.scheduled);
const withAll = (values: readonly string[], label: (v: string) => string) => [
  ...values.map((v) => ({ value: v, label: label(v) })),
  { value: "all", label: "All" },
];

/**
 * task を並べる page。Filter の小窓で status・area・project・scheduled を絞る。todo だけなら作った日順、
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
  const projects = useMemo(
    () =>
      [
        ...new Set(
          tasks
            .filter((t) => filters.area === "all" || t.area === filters.area)
            .flatMap((t) => (t.project ? [t.project] : [])),
        ),
      ].sort(),
    [tasks, filters.area],
  );
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
  const chips: FilterChip[] = [
    ...(filters.status !== "todo"
      ? [
          {
            label: filters.status === "all" ? "all statuses" : filters.status,
            onClear: () => set({ status: "todo" }),
          },
        ]
      : []),
    ...(filters.area !== "all"
      ? [{ label: filters.area, onClear: () => set({ area: "all", project: null }) }]
      : []),
    ...(filters.project ? [{ label: filters.project, onClear: () => set({ project: null }) }] : []),
    ...(filters.scheduled
      ? [{ label: "scheduled", onClear: () => set({ scheduled: false }) }]
      : []),
  ];

  const head = (
    <Filter chips={chips}>
      <Choice
        label="Status"
        data={withAll(STATUSES, (v) => STATUS_LABEL[v as keyof typeof STATUS_LABEL])}
        value={filters.status}
        onChange={(status) => set({ status })}
      />
      <Choice
        label="Area"
        data={[{ value: "all", label: "All" }, ...AREAS.map((a) => ({ value: a, label: a }))]}
        value={filters.area}
        onChange={(area) => set({ area, project: null })}
      />
      <Select
        label="Project"
        data={projects}
        value={filters.project}
        onChange={(project) => set({ project })}
      />
      <Switch
        label="Scheduled"
        checked={filters.scheduled}
        onChange={(scheduled) => set({ scheduled })}
      />
    </Filter>
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
          <Empty title="No task selected" description="Pick one from the list, or press ↓ / ↑." />
        )
      }
    />
  );
}
