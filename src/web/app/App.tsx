import { useState } from "react";

import { type RailItem, Shell } from "./Shell";
import { CreditsDialog } from "../components/CreditsDialog";
import { HelpDialog } from "../components/HelpDialog";
import { NewTaskDialog } from "../components/NewTaskDialog";
import {
  HomeIcon,
  ChecklistIcon,
  EmptyState,
  KeyboardIcon,
  MoonIcon,
  SeedlingIcon,
  Stack2Icon,
  SunIcon,
  useComputedColorScheme,
  useHotkeys,
  useMantineColorScheme,
} from "../components/ui";
import { type Page, useRoute } from "../hooks/useRoute";
import { useTasks } from "../hooks/useTasks";
import { Tasks } from "../pages/Tasks";
import { Home } from "../pages/Home";
import { Habits } from "../pages/Habits";

const PAGES: RailItem<Page>[] = [
  { key: "home", label: "Home", icon: HomeIcon },
  { key: "tasks", label: "Tasks", icon: ChecklistIcon },
  { key: "habits", label: "Habits", icon: SeedlingIcon },
];

export function App() {
  const { tasks, error, reload } = useTasks();
  const [{ page, id: selected }, go] = useRoute();
  const setPage = (p: Page) => go({ page: p, id: "" });
  // task を開く・閉じるは履歴に残さない
  const selectTask = (id: string) => go({ page: "tasks", id }, false);
  const [help, setHelp] = useState(false);
  const [credits, setCredits] = useState(false);
  const [adding, setAdding] = useState(false);
  // logo で origin に戻るたびに数を進め、Home を作り直して今日から見せる
  const [home, setHome] = useState(0);
  const goHome = () => {
    setPage("home");
    setHome((h) => h + 1);
  };
  const { setColorScheme } = useMantineColorScheme();
  const scheme = useComputedColorScheme("light");
  const flipTheme = () => setColorScheme(scheme === "dark" ? "light" : "dark");

  useHotkeys([
    ["shift+?", () => setHelp((h) => !h)],
    ["n", () => setAdding(true)],
  ]);

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
        { label: "Built with", icon: Stack2Icon, onClick: () => setCredits(true) },
      ]}
    >
      {error ? (
        <EmptyState title="Couldn't load tasks" description={error} />
      ) : !tasks ? (
        <EmptyState title="Loading…" />
      ) : page === "habits" ? (
        <Habits />
      ) : page === "tasks" ? (
        <Tasks
          tasks={tasks}
          selected={selected}
          onSelect={selectTask}
          onChanged={reload}
          onNew={() => setAdding(true)}
        />
      ) : (
        <Home key={home} tasks={tasks} onChanged={reload} onNew={() => setAdding(true)} />
      )}
      <NewTaskDialog opened={adding} onClose={() => setAdding(false)} onAdded={reload} />
      <HelpDialog opened={help} onClose={() => setHelp(false)} />
      <CreditsDialog opened={credits} onClose={() => setCredits(false)} />
    </Shell>
  );
}
