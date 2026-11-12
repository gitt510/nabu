import { type ReactNode, useState } from "react";

import { checkHabit, patchItem, type TaskView } from "../api";
import { STATUS_LABEL } from "../components/states";
import { ItemTree } from "../components/ItemTree";
import { TaskDetail } from "../components/TaskDetail";
import {
  Caption,
  Checkbox,
  Columns,
  EmptyState,
  ErrorText,
  Group,
  ListRow,
  Page,
  Pager,
  Section,
  Stack,
  useHotkeys,
  useMediaQuery,
} from "../components/ui";
import { useHabits } from "../hooks/useHabits";
import { useItems } from "../hooks/useItems";
import { childrenOf, isDone, leavesUnder } from "../outline";
import { addDays, dayLabel, dayOf, fmtDay, instant, isLate, timeOf, todayKey } from "../time";

const bySchedule = (a: TaskView, b: TaskView) =>
  instant(a.scheduled!).getTime() - instant(b.scheduled!).getTime();

/**
 * 時間で見る page。1 日分の予定を、時刻のあるものから順に並べる（日付だけの予定は終日）。
 * 終わった予定も、その日の記録として残す。今日か先の日を見ているときは、今日の時点で予定日を過ぎてまだ開いている
 * task を下に足す（今日の予定はまだ遅れていないので入れない）。過去の日には出さない。
 * 予定の下には毎日やる habit を並べ、その日にやった印をここで付け外しする。
 * その下には、見ている日の月が期日の item を木のまま並べ、印をここで付け外しする。
 * 広い画面では選んだ task を右の列に開き、狭い画面では Tasks page に移って開く
 */
export function Calendar({
  tasks,
  selected,
  onSelect,
  onOpen,
  onChanged,
}: {
  tasks: TaskView[];
  selected: string;
  onSelect: (id: string) => void;
  onOpen: (id: string) => void;
  onChanged: () => void;
}) {
  const [day, setDay] = useState(todayKey);
  const narrow = useMediaQuery("(max-width: 48em)") ?? false;
  const task = tasks.find((t) => String(t.id) === selected);
  const isToday = day === todayKey();
  const { habits, error, write } = useHabits(day);
  const outline = useItems();
  const month = day.slice(0, 7);
  const monthly = outline.items
    ? childrenOf(outline.items, null)
        .filter((i) => i.due?.startsWith(month))
        .sort((a, b) => a.due!.localeCompare(b.due!))
    : [];
  const planned = tasks.filter((t) => t.scheduled && dayOf(t.scheduled) === day).sort(bySchedule);
  const late =
    day >= todayKey()
      ? tasks
          .filter((t) => t.status === "todo" && t.scheduled && isLate(t.scheduled))
          .sort(bySchedule)
      : [];

  const prev = () => setDay((d) => addDays(d, -1));
  const next = () => setDay((d) => addDays(d, 1));
  useHotkeys([
    ["ArrowLeft", prev],
    ["ArrowRight", next],
  ]);

  const row = (t: TaskView, aside: ReactNode) => (
    <ListRow
      key={t.id}
      id={`day-${t.id}`}
      label={t.title}
      active={!narrow && String(t.id) === selected}
      onClick={() => (narrow ? onOpen : onSelect)(String(t.id))}
      aside={
        <Group gap={6} wrap="nowrap">
          {t.status === "todo" ? null : <Caption>{STATUS_LABEL[t.status]}</Caption>}
          {aside}
        </Group>
      }
    />
  );

  const day_ = (
    <Stack gap="lg">
      <Pager
        label={dayLabel(day)}
        prev={{ label: "Previous day (←)", onClick: prev }}
        next={{ label: "Next day (→)", onClick: next }}
        reset={isToday ? undefined : { label: "Today", onClick: () => setDay(todayKey()) }}
      />
      <Section title="Scheduled">
        {planned.length ? (
          <Stack gap={2}>
            {planned.map((t) =>
              row(t, <Caption mono>{timeOf(t.scheduled!) ?? "All day"}</Caption>),
            )}
          </Stack>
        ) : (
          <EmptyState title="Nothing scheduled" />
        )}
      </Section>
      {habits?.length ? (
        <Section
          title="Habits"
          aside={
            <Caption mono>
              {habits.filter((h) => h.done).length} / {habits.length}
            </Caption>
          }
        >
          <Stack gap="xs">
            {habits.map((h) => (
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
      ) : null}
      {monthly.length && outline.items ? (
        <Section
          title="This month"
          aside={
            <Caption mono>
              {monthly.filter((i) => isDone(outline.items!, i)).length} / {monthly.length}
            </Caption>
          }
        >
          <Stack gap="xs">
            <ItemTree
              items={outline.items}
              roots={monthly}
              row={(i) => (
                <Group justify="space-between" wrap="nowrap">
                  <Checkbox
                    label={i.text}
                    checked={isDone(outline.items!, i)}
                    onChange={(v) =>
                      void outline.write(
                        ...leavesUnder(outline.items!, i).map((l) => patchItem(l.id, { done: v })),
                      )
                    }
                  />
                  {i.due ? <Caption mono>{fmtDay(i.due)}</Caption> : null}
                </Group>
              )}
            />
            <ErrorText>{outline.error}</ErrorText>
          </Stack>
        </Section>
      ) : null}
      {late.length ? (
        <Section title="Overdue">
          <Stack gap={2}>
            {late.map((t) => row(t, <Caption mono>{fmtDay(t.scheduled!)}</Caption>))}
          </Stack>
        </Section>
      ) : null}
    </Stack>
  );
  if (narrow) return <Page>{day_}</Page>;
  return (
    <Page width="wide">
      <Columns>
        {day_}
        {task ? (
          <TaskDetail key={task.id} task={task} onChanged={onChanged} />
        ) : (
          <Section>
            <EmptyState title="No task selected" description="Pick one from the day." />
          </Section>
        )}
      </Columns>
    </Page>
  );
}
