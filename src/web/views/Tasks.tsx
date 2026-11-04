import { useMemo, useRef, useState } from "react";

import { AREAS } from "../../api/schema";
import type { Item } from "../api";
import { TaskDetail } from "../components/TaskDetail";
import { TaskList } from "../components/TaskList";
import { ArrowDownIcon, ArrowUpIcon, CalendarIcon, Caption, Empty, Group, HourglassIcon, IconButton, ListDetail, SearchInput, Select, Toggle, useHotkeys, useLocalStorage, useMediaQuery } from "../components/ui";
import { instant } from "../time";

/**
 * task を並べる page。Tasks は todo を作った日順に、Archive は done と dropped を直した日順に並べる（閉じた日は
 * 持たないので、直した日をその代わりにする）。area・project・scheduled・waiting で絞り、並び順は次に開いたときも残る
 */
export function Tasks({ archive, tasks, selected, onSelect, onChanged }: { archive: boolean; tasks: Item[]; selected: string; onSelect: (id: string, push?: boolean) => void; onChanged: () => void }) {
  const [view, setView] = useLocalStorage<{ oldest: boolean }>({ key: archive ? "nabu-archive-view" : "nabu-tasks-view", defaultValue: { oldest: false } });
  const [q, setQ] = useState("");
  const [area, setArea] = useState<string | null>(null);
  const [project, setProject] = useState<string | null>(null);
  const [scheduled, setScheduled] = useState(false);
  const [waiting, setWaiting] = useState(false);
  const search = useRef<HTMLInputElement>(null);
  const narrow = useMediaQuery("(max-width: 48em)") ?? false;

  const shown = useMemo(() => tasks.filter((t) => (t.status === "todo") !== archive), [tasks, archive]);
  const items = useMemo(() => {
    const words = q.trim().toLowerCase().split(/\s+/).filter(Boolean);
    const sign = view.oldest ? 1 : -1;
    const when = (t: Item) => instant(archive ? t.updated : t.created).getTime();
    return shown
      .filter((t) => words.every((w) => t.text.includes(w)) && (!area || t.area === area) && (!project || t.project === project) && (!scheduled || t.scheduled) && (!waiting || t.waiting))
      .sort((a, b) => sign * (when(a) - when(b)));
  }, [shown, q, area, project, scheduled, waiting, view.oldest, archive]);
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
  const flipOrder = () => setView({ ...view, oldest: !view.oldest });

  useHotkeys([
    ["/", () => search.current?.focus()],
    ["ArrowDown", () => step(1)],
    ["ArrowUp", () => step(-1)],
  ]);

  const head = (
    <>
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
            {items.length} / {shown.length}
          </Caption>
          <IconButton label={`Sorted by ${archive ? "updated" : "created"}, ${view.oldest ? "oldest" : "newest"} first`} icon={view.oldest ? ArrowUpIcon : ArrowDownIcon} onClick={flipOrder} />
        </Group>
      </Group>
    </>
  );

  return (
    <ListDetail
      head={head}
      list={<TaskList items={items} selected={selected} onSelect={open} archive={archive} />}
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
