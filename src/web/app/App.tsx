import { useState } from "react";

import { type RailItem, Shell } from "./Shell";
import { HelpDialog } from "../components/HelpDialog";
import {
  CalendarIcon,
  ChecklistIcon,
  EmptyState,
  KeyboardIcon,
  ListTreeIcon,
  MoonIcon,
  SeedlingIcon,
  SunIcon,
  useComputedColorScheme,
  useHotkeys,
  useMantineColorScheme,
} from "../components/ui";
import { useHash } from "../hooks/useHash";
import { useTasks } from "../hooks/useTasks";
import { Tasks } from "../pages/Tasks";
import { Calendar } from "../pages/Calendar";
import { Habits } from "../pages/Habits";
import { Items } from "../pages/Items";

type Page = "calendar" | "tasks" | "items" | "habits";
const PAGES: RailItem<Page>[] = [
  { key: "calendar", label: "Calendar", icon: CalendarIcon },
  { key: "tasks", label: "Tasks", icon: ChecklistIcon },
  { key: "items", label: "Items", icon: ListTreeIcon },
  { key: "habits", label: "Habits", icon: SeedlingIcon },
];

export function App() {
  const { tasks, error, reload } = useTasks();
  const [selected, setSelected] = useHash();
  // #番号で開いたときは Tasks、それ以外は今日から始める
  const [page, setPage] = useState<Page>(selected ? "tasks" : "calendar");
  const [help, setHelp] = useState(false);
  // logo で origin に戻るたびに数を進め、Calendar を作り直して今日から見せる
  const [home, setHome] = useState(0);
  const goHome = () => {
    setPage("calendar");
    setSelected("");
    setHome((h) => h + 1);
  };
  const { setColorScheme } = useMantineColorScheme();
  const scheme = useComputedColorScheme("light");
  const flipTheme = () => setColorScheme(scheme === "dark" ? "light" : "dark");

  useHotkeys([["shift+?", () => setHelp((h) => !h)]]);

  const openTask = (id: string) => {
    setPage("tasks");
    setSelected(id);
  };

  return (
    <Shell
      items={PAGES}
      current={page}
      onSelect={setPage}
      onHome={goHome}
      actions={[
        {
          label: scheme === "dark" ? "Light theme" : "Dark theme",
          icon: scheme === "dark" ? SunIcon : MoonIcon,
          onClick: flipTheme,
        },
        {
          label: "Keyboard shortcuts (?)",
          icon: KeyboardIcon,
          onClick: () => setHelp(true),
          keyboard: true,
        },
      ]}
    >
      {error ? (
        <EmptyState title="Couldn't load tasks" description={error} />
      ) : !tasks ? (
        <EmptyState title="Loading…" />
      ) : page === "habits" ? (
        <Habits />
      ) : page === "items" ? (
        <Items />
      ) : page === "tasks" ? (
        <Tasks tasks={tasks} selected={selected} onSelect={setSelected} onChanged={reload} />
      ) : (
        <Calendar
          key={home}
          tasks={tasks}
          selected={selected}
          onSelect={setSelected}
          onOpen={openTask}
          onChanged={reload}
        />
      )}
      <HelpDialog opened={help} onClose={() => setHelp(false)} />
    </Shell>
  );
}
