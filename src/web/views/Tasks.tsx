import { useMemo, useRef, useState } from "react";

import { AREAS } from "../../api/schema";
import { check, type Item } from "../api";
import { TaskDetail } from "../components/TaskDetail";
import { TaskList } from "../components/TaskList";
import { ArrowDownIcon, ArrowUpIcon, CalendarIcon, Caption, Empty, ErrorText, Group, HourglassIcon, IconButton, ListDetail, SearchInput, Segments, Select, Toggle, useHotkeys, useLocalStorage, useMediaQuery } from "../components/ui";
import { instant } from "../time";

// Done には完了と dropped の両方が並ぶ
type Tab = "todo" | "done";
const TABS: { value: Tab; label: string }[] = [
  { value: "todo", label: "Todo" },
  { value: "done", label: "Done" },
];

/**
 * 状態で見る page。まだか閉じたかを Tabs で選び、area・project・scheduled・waiting はそれに掛け合わせて絞る。
 * 選んだ tab と並び順は次に開いたときも残る
 */
export function Tasks({ tasks, selected, onSelect, onChanged }: { tasks: Item[]; selected: string; onSelect: (id: string, push?: boolean) => void; onChanged: () => void }) {
  const [view, setView] = useLocalStorage<{ tab: Tab; oldest: boolean }>({ key: "nabu-tasks-view", defaultValue: { tab: "todo", oldest: false } });
  // 前の版が残した tab（all、inbox…）は Todo として読む
  const tab: Tab = view.tab === "done" ? "done" : "todo";
  const [error, setError] = useState<string>();
  const [q, setQ] = useState("");
  const [area, setArea] = useState<string | null>(null);
  const [project, setProject] = useState<string | null>(null);
  const [scheduled, setScheduled] = useState(false);
  const [waiting, setWaiting] = useState(false);
  const search = useRef<HTMLInputElement>(null);
  const narrow = useMediaQuery("(max-width: 48em)") ?? false;

  const inTab = useMemo(() => tasks.filter((t) => (t.status === "todo") === (tab === "todo")), [tasks, tab]);
  const items = useMemo(() => {
    const words = q.trim().toLowerCase().split(/\s+/).filter(Boolean);
    const sign = view.oldest ? 1 : -1;
    return inTab
      .filter((t) => words.every((w) => t.text.includes(w)) && (!area || t.area === area) && (!project || t.project === project) && (!scheduled || t.scheduled) && (!waiting || t.waiting))
      .sort((a, b) => sign * (instant(a.created).getTime() - instant(b.created).getTime()));
  }, [inTab, q, area, project, scheduled, waiting, view.oldest]);
  const projects = useMemo(() => [...new Set(tasks.filter((t) => !area || t.area === area).flatMap((t) => (t.project ? [t.project] : [])))].sort(), [tasks, area]);
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
  const stepTab = (delta: number) => {
    const at = TABS.findIndex((t) => t.value === tab);
    setView({ ...view, tab: TABS[(at + delta + TABS.length) % TABS.length].value });
  };
  const mark = async (id: number, done: boolean) => {
    const problem = await check(id, done);
    setError(problem);
    if (!problem) onChanged();
  };
  const flipOrder = () => setView({ ...view, oldest: !view.oldest });

  useHotkeys([
    ["/", () => search.current?.focus()],
    ["ArrowDown", () => step(1)],
    ["ArrowUp", () => step(-1)],
    ["[", () => stepTab(-1)],
    ["]", () => stepTab(1)],
  ]);

  const head = (
    <>
      <Segments items={TABS} value={tab} onChange={(tab) => setView({ ...view, tab })} />
      <SearchInput ref={search} value={q} onChange={setQ} />
      <Group gap="xs" wrap="nowrap">
        <Select label="area" data={[...AREAS]} value={area} onChange={(a) => { setArea(a); setProject(null); }} />
        <Select label="project" data={projects} value={project} onChange={setProject} />
      </Group>
      <Group justify="space-between">
        <Group gap={6}>
          <Toggle label="scheduled" icon={CalendarIcon} checked={scheduled} onChange={setScheduled} />
          <Toggle label="waiting" icon={HourglassIcon} checked={waiting} onChange={setWaiting} />
        </Group>
        <Group gap={4}>
          <Caption mono>
            {items.length} / {inTab.length}
          </Caption>
          <IconButton label={`Sorted by created, ${view.oldest ? "oldest" : "newest"} first`} icon={view.oldest ? ArrowUpIcon : ArrowDownIcon} onClick={flipOrder} />
        </Group>
      </Group>
      <ErrorText>{error}</ErrorText>
    </>
  );

  return (
    <ListDetail
      head={head}
      list={<TaskList items={items} selected={selected} onSelect={open} onCheck={(t, done) => void mark(t.id, done)} />}
      showDetail={!!task}
      detail={
        task ? (
          <TaskDetail key={task.id} task={task} onBack={narrow ? () => onSelect("") : undefined} onChanged={onChanged} />
        ) : (
          <Empty title="No task selected" description="Pick one from the list, or press ↓ / ↑." />
        )
      }
    />
  );
}
