import { useEffect, useState } from "react";
import {
  type ColumnFiltersState,
  columnFacetingFeature,
  columnFilteringFeature,
  createColumnHelper,
  createFacetedRowModel,
  createFacetedUniqueValues,
  createFilteredRowModel,
  createSortedRowModel,
  filterFn_arrHas,
  filterFn_equals,
  rowExpandingFeature,
  rowSortingFeature,
  sortFn_alphanumeric,
  sortFn_basic,
  tableFeatures,
  useTable,
} from "@tanstack/react-table";

import { AREAS } from "../../api/schema";
import type { TaskView } from "../api";
import { STATUS_LABEL } from "../components/states";
import { TaskDetail } from "../components/TaskDetail";
import { TaskMenu } from "../components/TaskMenu";
import {
  Button,
  Caption,
  CheckFacet,
  EmptyState,
  Facet,
  FoldTable,
  Page,
  PageTitle,
  PlusIcon,
  SideColumn,
  Stack,
  useHotkeys,
} from "../components/ui";
import { fmtDay, instant } from "../time";

// 絞り込み・数え上げ・並べ替え・開閉は TanStack Table に任せ、描くのは FoldTable
const features = tableFeatures({
  columnFilteringFeature,
  filteredRowModel: createFilteredRowModel(),
  filterFns: { equals: filterFn_equals, arrHas: filterFn_arrHas },
  columnFacetingFeature,
  facetedRowModel: createFacetedRowModel(),
  facetedUniqueValues: createFacetedUniqueValues(),
  rowSortingFeature,
  sortedRowModel: createSortedRowModel(),
  sortFns: { alphanumeric: sortFn_alphanumeric, basic: sortFn_basic },
  rowExpandingFeature,
});

/** open は todo、closed は done と canceled */
const stateOf = (t: TaskView) => (t.status === "todo" ? "open" : "closed");
const time = (iso: string | null) => (iso ? instant(iso).getTime() : undefined);

const helper = createColumnHelper<typeof features, TaskView>();
const columns = helper.columns([
  helper.accessor("title", { id: "title", sortFn: "alphanumeric" }),
  helper.accessor((t) => t.project ?? undefined, {
    id: "project",
    sortFn: "alphanumeric",
    sortUndefined: "last",
  }),
  // area は選んだもののどれかに当たれば残し、何も選んでいなければ絞らない
  helper.accessor("area", { id: "area", filterFn: "arrHas", sortFn: "alphanumeric" }),
  helper.accessor((t) => time(t.scheduled), {
    id: "scheduled",
    sortFn: "basic",
    sortUndefined: "last",
  }),
  // 開いている task は作った日、閉じた task は直した日（閉じた日は持たないので、その代わり）
  helper.accessor((t) => time(t.status === "todo" ? t.created : t.updated), {
    id: "when",
    sortFn: "basic",
    sortDescFirst: true,
  }),
  helper.accessor(stateOf, { id: "state", filterFn: "equals" }),
  // 絞り込みだけに使う列。予定があるか、どの project か（"" は project なし）
  helper.accessor((t) => (t.scheduled ? "scheduled" : "unscheduled"), {
    id: "hasSchedule",
    filterFn: "arrHas",
  }),
  helper.accessor((t) => t.project ?? "", { id: "projectIs", filterFn: "arrHas" }),
]);

/**
 * task を並べる page。表の行を押すとその場で開き、もう一度押すと閉じる。開くのは 1 つだけで、URL に残す。
 * 見出しを押すと並べ替え、最初は新しいものから。
 * 左の列の Facet で、開いている task と閉じた task、予定の有無・area・project を絞る（それぞれに、ほかの軸の絞り込みを効かせた数を添える）
 */
export function Tasks({
  tasks,
  selected,
  onSelect,
  onChanged,
  onNew,
}: {
  tasks: TaskView[];
  selected: string;
  onSelect: (id: string) => void;
  onChanged: () => void;
  onNew: () => void;
}) {
  // URL が閉じた task を指していれば、閉じた側から見せる
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>(() => {
    const t = tasks.find((t) => String(t.id) === selected);
    return [{ id: "state", value: t ? stateOf(t) : "open" }];
  });
  const table = useTable({
    features,
    columns,
    data: tasks,
    getRowId: (t) => String(t.id),
    getRowCanExpand: () => true,
    // 開いている行は URL が持つ。開けるのは 1 つだけ
    state: { columnFilters, expanded: selected ? { [selected]: true } : {} },
    onColumnFiltersChange: setColumnFilters,
    onExpandedChange: (updater) => {
      const prev = selected ? { [selected]: true } : {};
      const next = typeof updater === "function" ? updater(prev) : updater;
      const ids = next === true ? [] : Object.keys(next).filter((id) => next[id]);
      open(ids.find((id) => id !== selected) ?? "");
    },
    initialState: { sorting: [{ id: "when", desc: true }] },
  });
  const rows = table.getRowModel().rows;
  const state = table.getColumn("state")!;
  const closed = state.getFilterValue() === "closed";
  // project の選択肢は、絞り込みで消えないように全 task から取る。project なしは最後に
  const projects = [...new Set(tasks.map((t) => t.project ?? ""))].sort((a, b) =>
    a === "" ? 1 : b === "" ? -1 : a.localeCompare(b),
  );

  // いくつも選べる軸。何も選んでいなければ絞らない
  const multi = (id: string, title: string, options: { value: string; label: string }[]) => {
    const column = table.getColumn(id)!;
    return (
      <CheckFacet
        title={title}
        data={options.map((o) => ({
          ...o,
          count: column.getFacetedUniqueValues().get(o.value) ?? 0,
        }))}
        value={(column.getFilterValue() as string[] | undefined) ?? []}
        onChange={(value) => column.setFilterValue(value)}
      />
    );
  };
  // 狭い画面で畳んだときに見せる、今の絞り込み
  const summary = [
    closed ? "Closed" : "Open",
    ...columnFilters
      .filter((f) => f.id !== "state")
      .map((f) => (f.value as string[]).map((v) => v || "no project").join(", ")),
    String(rows.length),
  ].join(" · ");

  function open(id: string) {
    onSelect(id);
    // 開いた行が見えるように。閉じる行の高さが縮みきってから
    if (id) setTimeout(() => scrollTo(id), 50);
  }
  const step = (delta: number) => {
    const at = rows.findIndex((r) => r.id === selected);
    const next = rows[Math.min(rows.length - 1, Math.max(0, at + delta))];
    if (next) open(next.id);
  };

  useHotkeys([
    ["ArrowDown", () => step(1)],
    ["ArrowUp", () => step(-1)],
  ]);
  // URL から開いたときは、最初にその行まで送る
  const [first] = useState(selected);
  useEffect(() => {
    if (first) scrollTo(first);
  }, [first]);

  const facets = (
    <>
      <Facet
        title="State"
        data={(["open", "closed"] as const).map((value) => ({
          value,
          label: value === "open" ? "Open" : "Closed",
          count: state.getFacetedUniqueValues().get(value) ?? 0,
        }))}
        value={state.getFilterValue() as string}
        onChange={(value) => state.setFilterValue(value)}
      />
      {multi("hasSchedule", "Schedule", [
        { value: "scheduled", label: "Scheduled" },
        { value: "unscheduled", label: "Unscheduled" },
      ])}
      {multi(
        "area",
        "Area",
        AREAS.map((a) => ({ value: a, label: a })),
      )}
      {multi(
        "projectIs",
        "Project",
        projects.map((p) => ({ value: p, label: p || "No project" })),
      )}
    </>
  );

  const sortable = (id: string, header: string, extra: { end?: boolean; keep?: boolean } = {}) => {
    const column = table.getColumn(id)!;
    return {
      id,
      header,
      sorted: column.getIsSorted(),
      onSort: column.getToggleSortingHandler(),
      ...extra,
    };
  };

  return (
    <SideColumn side={facets} summary={summary}>
      <Page>
        <Stack gap="md">
          <PageTitle
            aside={
              <Button icon={PlusIcon} onClick={onNew}>
                New task
              </Button>
            }
          >
            Tasks
          </PageTitle>
          {rows.length ? (
            <FoldTable
              columns={[
                sortable("title", "Title", { keep: true }),
                sortable("scheduled", "Scheduled"),
                sortable("area", "Area"),
                sortable("project", "Project"),
                sortable("when", closed ? "Updated" : "Created", { end: true, keep: true }),
              ]}
              rows={rows.map((r) => {
                const t = r.original;
                return {
                  id: r.id,
                  anchor: `task-${r.id}`,
                  open: r.getIsExpanded(),
                  onToggle: () => r.toggleExpanded(),
                  cells: {
                    title: (
                      <>
                        {t.title} <Caption mono>#{t.id}</Caption>
                      </>
                    ),
                    project: t.project ? <Caption>{t.project}</Caption> : null,
                    area: <Caption>{t.area}</Caption>,
                    scheduled: t.scheduled ? <Caption mono>{fmtDay(t.scheduled)}</Caption> : null,
                    when: closed ? (
                      <Caption>
                        {STATUS_LABEL[t.status]} <Caption mono>{fmtDay(t.updated)}</Caption>
                      </Caption>
                    ) : (
                      <Caption mono>{fmtDay(t.created)}</Caption>
                    ),
                  },
                  actions: <TaskMenu task={t} onChanged={onChanged} />,
                  detail: <TaskDetail task={t} onChanged={onChanged} />,
                };
              })}
            />
          ) : (
            <EmptyState title="No tasks" description="Try other filters." />
          )}
        </Stack>
      </Page>
    </SideColumn>
  );
}

const scrollTo = (id: string) =>
  document.getElementById(`task-${id}`)?.scrollIntoView({ block: "nearest", behavior: "smooth" });
