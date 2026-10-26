import { useMemo, useRef, useState } from "react";

import { AREAS, type Status } from "../../src/fields";
import type { Item } from "../api";
import { TaskDetail } from "../components/TaskDetail";
import { TaskList } from "../components/TaskList";
import { Caption, CalendarIcon, DownIcon, Empty, Group, IconButton, ListDetail, SearchInput, Segments, Select, Toggle, UpIcon, useHotkeys, useLocalStorage, useMediaQuery, WaitingIcon } from "../components/ui";
import { instant } from "../time";

type Tab = "all" | Status;
const TABS: { value: Tab; label: string }[] = [
  { value: "all", label: "All" },
  { value: "doing", label: "Doing" },
  { value: "inbox", label: "Inbox" },
  { value: "done", label: "Done" },
  { value: "dropped", label: "Dropped" },
];

/**
 * 状態で見る page。status は Tabs で 1 つ選び、area・project・scheduled・waiting はそれに掛け合わせて絞る。
 * 選んだ tab と並び順は次に開いたときも残る
 */
export function Tasks({ tasks, selected, onSelect, onChanged }: { tasks: Item[]; selected: string; onSelect: (slug: string, push?: boolean) => void; onChanged: () => void }) {
  const [view, setView] = useLocalStorage<{ tab: Tab; oldest: boolean }>({ key: "nabu-tasks-view", defaultValue: { tab: "all", oldest: false } });
  const [q, setQ] = useState("");
  const [area, setArea] = useState<string | null>(null);
  const [project, setProject] = useState<string | null>(null);
  const [scheduled, setScheduled] = useState(false);
  const [waiting, setWaiting] = useState(false);
  const [menuOpened, setMenuOpened] = useState(false);
  const search = useRef<HTMLInputElement>(null);
  const narrow = useMediaQuery("(max-width: 48em)") ?? false;

  const inTab = useMemo(() => tasks.filter((t) => view.tab === "all" || t.status === view.tab), [tasks, view.tab]);
  const items = useMemo(() => {
    const words = q.trim().toLowerCase().split(/\s+/).filter(Boolean);
    const sign = view.oldest ? 1 : -1;
    return inTab
      .filter((t) => words.every((w) => t.text.includes(w)) && (!area || t.area === area) && (!project || t.project === project) && (!scheduled || t.scheduled) && (!waiting || t.waiting))
      .sort((a, b) => sign * (instant(a.created).getTime() - instant(b.created).getTime()));
  }, [inTab, q, area, project, scheduled, waiting, view.oldest]);
  const projects = useMemo(() => [...new Set(tasks.filter((t) => !area || t.area === area).flatMap((t) => (t.project ? [t.project] : [])))].sort(), [tasks, area]);
  const task = tasks.find((t) => t.slug === selected);

  const open = (slug: string) => {
    onSelect(slug, narrow);
    document.getElementById(`task-${slug}`)?.scrollIntoView({ block: "nearest" });
  };
  const step = (delta: number) => {
    const at = items.findIndex((t) => t.slug === selected);
    const next = items[Math.min(items.length - 1, Math.max(0, at + delta))];
    if (next) open(next.slug);
  };
  const stepTab = (delta: number) => {
    const at = TABS.findIndex((t) => t.value === view.tab);
    setView({ ...view, tab: TABS[(at + delta + TABS.length) % TABS.length].value });
  };
  const flipOrder = () => setView({ ...view, oldest: !view.oldest });

  useHotkeys([
    ["/", () => search.current?.focus()],
    ["j", () => step(1)],
    ["k", () => step(-1)],
    ["[", () => stepTab(-1)],
    ["]", () => stepTab(1)],
    ["s", flipOrder],
    ["m", () => task && setMenuOpened((o) => !o)],
  ]);

  const head = (
    <>
      <Segments items={TABS} value={view.tab} onChange={(tab) => setView({ ...view, tab })} />
      <SearchInput ref={search} value={q} onChange={setQ} />
      <Group gap="xs" wrap="nowrap">
        <Select label="area" data={[...AREAS]} value={area} onChange={(a) => { setArea(a); setProject(null); }} />
        <Select label="project" data={projects} value={project} onChange={setProject} />
      </Group>
      <Group justify="space-between">
        <Group gap={6}>
          <Toggle label="scheduled" icon={CalendarIcon} checked={scheduled} onChange={setScheduled} />
          <Toggle label="waiting" icon={WaitingIcon} checked={waiting} onChange={setWaiting} />
        </Group>
        <Group gap={4}>
          <Caption mono>
            {items.length} / {inTab.length}
          </Caption>
          <IconButton label={`Sorted by created, ${view.oldest ? "oldest" : "newest"} first (s)`} icon={view.oldest ? UpIcon : DownIcon} onClick={flipOrder} />
        </Group>
      </Group>
    </>
  );

  return (
    <ListDetail
      head={head}
      list={<TaskList items={items} selected={selected} onSelect={open} />}
      showDetail={!!task}
      detail={
        task ? (
          <TaskDetail key={task.slug} task={task} onBack={narrow ? () => onSelect("") : undefined} onChanged={onChanged} menuOpened={menuOpened} onMenuOpenedChange={setMenuOpened} />
        ) : (
          <Empty title="No task selected" description="Pick one from the list, or press j / k." />
        )
      }
    />
  );
}
