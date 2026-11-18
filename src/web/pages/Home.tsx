import { useState } from "react";

import { checkHabit, type TaskView } from "../api";
import { TaskDetail } from "../components/TaskDetail";
import { TaskTitle, TaskWhen } from "../components/TaskTitle";
import {
  Caption,
  Checkbox,
  EmptyState,
  ErrorText,
  FoldRow,
  Page,
  Pager,
  Section,
  Stack,
  useHotkeys,
} from "../components/ui";
import { useHabits } from "../hooks/useHabits";
import { addDays, dayLabel, instant, todayKey } from "../time";

/** Home に出す task の数 */
const LATEST = 5;

/**
 * 最初に開く page。Tasks の枠に、開いている task を作った日の新しい順に LATEST 件まで並べる（見ている日によらない）。
 * Tasks の下には毎日やる habit を（まだ無くても枠ごと）並べ、その日にやった印をここで付け外しする。
 * 行を押すと task をその場で開く（開くのは 1 つだけで、URL には残さない）
 */
export function Home({ tasks, onChanged }: { tasks: TaskView[]; onChanged: () => void }) {
  const [day, setDay] = useState(todayKey);
  const [selected, setSelected] = useState("");
  const isToday = day === todayKey();
  const { habits, error, write } = useHabits(day);
  const latest = tasks
    .filter((t) => t.status === "todo")
    .sort((a, b) => instant(b.created).getTime() - instant(a.created).getTime())
    .slice(0, LATEST);

  const prev = () => setDay((d) => addDays(d, -1));
  const next = () => setDay((d) => addDays(d, 1));
  useHotkeys([
    ["ArrowLeft", prev],
    ["ArrowRight", next],
  ]);

  return (
    <Page>
      <Stack gap="lg">
        <Pager
          label={dayLabel(day)}
          prev={{ label: "Previous day (←)", onClick: prev }}
          next={{ label: "Next day (→)", onClick: next }}
          reset={isToday ? undefined : { label: "Today", onClick: () => setDay(todayKey()) }}
        />
        <Section title="Tasks">
          {latest.length ? (
            <Stack gap={2}>
              {latest.map((t) => (
                <FoldRow
                  key={t.id}
                  id={`day-${t.id}`}
                  label={<TaskTitle task={t} />}
                  open={String(t.id) === selected}
                  onToggle={() => setSelected(String(t.id) === selected ? "" : String(t.id))}
                  aside={<TaskWhen task={t} />}
                >
                  <TaskDetail task={t} onChanged={onChanged} />
                </FoldRow>
              ))}
            </Stack>
          ) : (
            <EmptyState title="Nothing to do" />
          )}
        </Section>
        <Section
          title="Habits"
          aside={
            habits?.length ? (
              <Caption mono>
                {habits.filter((h) => h.done).length} / {habits.length}
              </Caption>
            ) : null
          }
        >
          <Stack gap="xs">
            {habits && !habits.length ? (
              <EmptyState title="No habits" description="Add one on the Habits page." />
            ) : null}
            {habits?.map((h) => (
              <Checkbox
                key={h.id}
                label={h.title}
                checked={h.done}
                onChange={(v) => void write(checkHabit(h.id, day, v))}
              />
            ))}
            <ErrorText>{error}</ErrorText>
          </Stack>
        </Section>
      </Stack>
    </Page>
  );
}
