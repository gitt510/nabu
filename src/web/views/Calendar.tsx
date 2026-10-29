import { type ReactNode, useState } from "react";

import { checkHabit, type Item } from "../api";
import { STATUS_CHOICES } from "../components/statuses";
import { TaskDetail } from "../components/TaskDetail";
import { Caption, Checkbox, Columns, Empty, ErrorText, Group, ListRow, Page, Pager, Section, Stack, Tag, useHotkeys, useMediaQuery } from "../components/ui";
import { useHabits } from "../hooks/useHabits";
import { addDays, dayLabel, dayOf, fmtDay, instant, isLate, timeOf, todayKey } from "../time";

const open_ = (t: Item) => t.status !== "done" && t.status !== "dropped";
const bySchedule = (a: Item, b: Item) => instant(a.scheduled!).getTime() - instant(b.scheduled!).getTime();

/**
 * 時間で見る page。1 日分の予定を、時刻のあるものから順に並べる（日付だけの予定は終日）。
 * 終わった予定も、その日の記録として残す。今日か先の日を見ているときは、今日の時点で予定日を過ぎてまだ開いている
 * task を下に足す（今日の予定はまだ遅れていないので入れない）。過去の日には出さない。
 * 予定の下には毎日やる habit を並べ、その日にやった印をここで付け外しする。
 * 広い画面では選んだ task を右の列に開き、狭い画面では Tasks page に移って開く
 */
export function Calendar({ tasks, selected, onSelect, onOpen, onChanged }: { tasks: Item[]; selected: string; onSelect: (slug: string) => void; onOpen: (slug: string) => void; onChanged: () => void }) {
  const [day, setDay] = useState(todayKey);
  const [menuOpened, setMenuOpened] = useState(false);
  const narrow = useMediaQuery("(max-width: 48em)") ?? false;
  const task = tasks.find((t) => t.slug === selected);
  const isToday = day === todayKey();
  const { habits, error, write } = useHabits(day);
  const planned = tasks.filter((t) => t.scheduled && dayOf(t.scheduled) === day).sort(bySchedule);
  const late = day >= todayKey() ? tasks.filter((t) => open_(t) && t.scheduled && isLate(t.scheduled)).sort(bySchedule) : [];

  const prev = () => setDay((d) => addDays(d, -1));
  const next = () => setDay((d) => addDays(d, 1));
  useHotkeys([
    ["ArrowLeft", prev],
    ["ArrowRight", next],
    ["0", () => setDay(todayKey())],
  ]);

  const row = (t: Item, aside: ReactNode) => (
    <ListRow
      key={t.slug}
      id={`day-${t.slug}`}
      label={t.title}
      active={!narrow && t.slug === selected}
      onClick={() => (narrow ? onOpen : onSelect)(t.slug)}
      aside={
        <Group gap={6} wrap="nowrap">
          {open_(t) ? null : <Tag tone="muted">{STATUS_CHOICES.find((c) => c.value === t.status)?.label}</Tag>}
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
          reset={isToday ? undefined : { label: "Today (0)", onClick: () => setDay(todayKey()) }}
        />
        <Section title="Scheduled">
          {planned.length ? (
            <Stack gap={2}>{planned.map((t) => row(t, <Caption mono>{timeOf(t.scheduled!) ?? "All day"}</Caption>))}</Stack>
          ) : (
            <Empty title="Nothing scheduled" />
          )}
        </Section>
        {habits?.length ? (
          <Section title="Habits" aside={<Caption mono>{habits.filter((h) => h.done).length} / {habits.length}</Caption>}>
            <Stack gap="xs">
              {habits.map((h) => (
                <Checkbox key={h.id} label={h.title} checked={h.done} onChange={(v) => void write(checkHabit(h.id, day, v))} />
              ))}
              <ErrorText>{error}</ErrorText>
            </Stack>
          </Section>
        ) : null}
        {late.length ? (
          <Section title="Overdue">
            <Stack gap={2}>{late.map((t) => row(t, <Tag tone="late">{fmtDay(t.scheduled!)}</Tag>))}</Stack>
          </Section>
        ) : null}
    </Stack>
  );
  if (narrow) return <Page>{day_}</Page>;
  return (
    <Page width="wide">
      <Columns>
        {day_}
        <Section>
          {task ? (
            <TaskDetail key={task.slug} task={task} onChanged={onChanged} menuOpened={menuOpened} onMenuOpenedChange={setMenuOpened} />
          ) : (
            <Empty title="No task selected" description="Pick one from the day." />
          )}
        </Section>
      </Columns>
    </Page>
  );
}
