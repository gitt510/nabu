import { type ReactNode, useState } from "react";

import type { Item } from "../api";
import { STATUS_CHOICES } from "../components/statuses";
import { Caption, Empty, Group, ListRow, Page, Pager, Stack, Subheading, Tag, useHotkeys } from "../components/ui";
import { addDays, dayLabel, dayOf, fmtDay, instant, isLate, timeOf, todayKey } from "../time";

const open_ = (t: Item) => t.status !== "done" && t.status !== "dropped";
const bySchedule = (a: Item, b: Item) => instant(a.scheduled!).getTime() - instant(b.scheduled!).getTime();

/**
 * 時間で見る page。1 日分の予定を、時刻のあるものから順に並べる（日付だけの予定は終日）。
 * 終わった予定も、その日の記録として残す。今日を見ているときだけ、予定日を過ぎてまだ開いている task を下に足す
 */
export function Calendar({ tasks, onOpen }: { tasks: Item[]; onOpen: (slug: string) => void }) {
  const [day, setDay] = useState(todayKey);
  const isToday = day === todayKey();
  const planned = tasks.filter((t) => t.scheduled && dayOf(t.scheduled) === day).sort(bySchedule);
  const late = isToday ? tasks.filter((t) => open_(t) && t.scheduled && isLate(t.scheduled)).sort(bySchedule) : [];

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
      active={false}
      onClick={() => onOpen(t.slug)}
      aside={
        <Group gap={6} wrap="nowrap">
          {open_(t) ? null : <Tag tone="muted">{STATUS_CHOICES.find((c) => c.value === t.status)?.label}</Tag>}
          {aside}
        </Group>
      }
    />
  );

  return (
    <Page>
      <Stack gap="lg">
        <Pager
          label={dayLabel(day)}
          prev={{ label: "Previous day (←)", onClick: prev }}
          next={{ label: "Next day (→)", onClick: next }}
          reset={isToday ? undefined : { label: "Today (0)", onClick: () => setDay(todayKey()) }}
        />
        {planned.length ? (
          <Stack gap={2}>{planned.map((t) => row(t, <Caption mono>{timeOf(t.scheduled!) ?? "All day"}</Caption>))}</Stack>
        ) : (
          <Empty title="Nothing scheduled" />
        )}
        {late.length ? (
          <Stack gap={0}>
            <Subheading>Overdue</Subheading>
            <Stack gap={2}>{late.map((t) => row(t, <Tag tone="late">{fmtDay(t.scheduled!)}</Tag>))}</Stack>
          </Stack>
        ) : null}
      </Stack>
    </Page>
  );
}
