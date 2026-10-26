import { useState } from "react";

import { HelpDialog } from "./components/HelpDialog";
import { CalendarIcon, DarkIcon, Empty, KeysIcon, LightIcon, type RailItem, Shell, TasksIcon, useComputedColorScheme, useHotkeys, useMantineColorScheme, WatchingIcon } from "./components/ui";
import { useHash } from "./hooks/useHash";
import { useTasks } from "./hooks/useTasks";
import { Tasks } from "./views/Tasks";
import { Calendar } from "./views/Calendar";

type Page = "calendar" | "tasks" | "watching";
const PAGES: RailItem<Page>[] = [
  { key: "calendar", label: "Calendar", icon: CalendarIcon },
  { key: "tasks", label: "Tasks", icon: TasksIcon },
  { key: "watching", label: "Watching", icon: WatchingIcon, disabled: true },
];

export function App() {
  const { tasks, error, reload } = useTasks();
  const [slug, setSlug] = useHash();
  // #slug で開いたときは Tasks、それ以外は今日から始める
  const [page, setPage] = useState<Page>(slug ? "tasks" : "calendar");
  const [help, setHelp] = useState(false);
  // logo で origin に戻るたびに数を進め、Calendar を作り直して今日から見せる
  const [home, setHome] = useState(0);
  const goHome = () => {
    setPage("calendar");
    setSlug("");
    setHome((h) => h + 1);
  };
  const { setColorScheme } = useMantineColorScheme();
  const scheme = useComputedColorScheme("light");
  const flipTheme = () => setColorScheme(scheme === "dark" ? "light" : "dark");

  useHotkeys([
    ["t", flipTheme],
    ["shift+?", () => setHelp((h) => !h)],
  ]);

  const openTask = (s: string) => {
    setPage("tasks");
    setSlug(s);
  };

  return (
    <Shell
      items={PAGES}
      current={page}
      onSelect={setPage}
      onHome={goHome}
      actions={[
        { label: scheme === "dark" ? "Light theme (t)" : "Dark theme (t)", icon: scheme === "dark" ? LightIcon : DarkIcon, onClick: flipTheme },
        { label: "Keyboard shortcuts (?)", icon: KeysIcon, onClick: () => setHelp(true) },
      ]}
    >
      {error ? (
        <Empty title="Couldn't load tasks" description={error} />
      ) : !tasks ? (
        <Empty title="Loading…" />
      ) : page === "tasks" ? (
        <Tasks tasks={tasks} selected={slug} onSelect={setSlug} onChanged={reload} />
      ) : (
        <Calendar key={home} tasks={tasks} onOpen={openTask} />
      )}
      <HelpDialog opened={help} onClose={() => setHelp(false)} />
    </Shell>
  );
}
