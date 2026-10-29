import { useState } from "react";

import type { Status, Task } from "../../api/schema";
import { type Item, patch } from "../api";
import { dayOf, fmt, fmtDay, isLate, ticket, timeOf, todayKey } from "../time";
import { DropDialog } from "./DropDialog";
import { STATUS_CHOICES } from "./statuses";
import { ArrowLeftIcon, Caption, ChoiceMenu, CirclePlusIcon, CopyText, DateMenu, ErrorText, ExternalLink, GitPullRequestIcon, Group, Heading, IconButton, type IconComponent, LinkIcon, PencilIcon, Prose, Section, Stack, Stamp, Tag, TicketIcon } from "./ui";

/** 開いている task：番号を添えた題と右端の status、その下に properties と作った日・直した日の 1 行と link の 1 行、枠の中に本文。予定日以外の値のない property は出さない */
export function TaskDetail({ task, onBack, onChanged, menuOpened, onMenuOpenedChange }: { task: Item; onBack?: () => void; onChanged: () => void; menuOpened: boolean; onMenuOpenedChange: (o: boolean) => void }) {
  const [error, setError] = useState<string>();
  const [dropping, setDropping] = useState(false);

  const save = async (fields: Partial<Task>) => {
    const problem = await patch(task.id, fields);
    setError(problem);
    if (!problem) onChanged();
  };

  const move = (status: Status) => {
    if (status === task.status) return;
    if (status === "dropped") return setDropping(true);
    save({ status });
  };

  // 時刻のある予定は、時刻を残して日だけを移す
  const schedule = (day: string | null) => {
    const time = task.scheduled && timeOf(task.scheduled);
    save({ scheduled: day && time ? `${day}T${time}:00+09:00` : day });
  };

  const links = (urls: string[], icon: IconComponent) =>
    urls.map((u) => (
      <ExternalLink key={u} href={u} icon={icon}>
        {ticket(u)}
      </ExternalLink>
    ));

  return (
    <Stack gap="md">
      {onBack ? (
        <Group>
          <IconButton label="Back to list" icon={ArrowLeftIcon} onClick={onBack} />
        </Group>
      ) : null}
      <Stack gap="xs">
        <Heading aside={<ChoiceMenu choices={STATUS_CHOICES} value={task.status} onSelect={move} opened={menuOpened} onOpenedChange={onMenuOpenedChange} />}>
          {task.title} <CopyText value={`#${task.id}`} />
        </Heading>
        <Group gap="xs">
          <Tag>{task.area}</Tag>
          {task.project ? <Tag>{task.project}</Tag> : null}
          <DateMenu
            label={task.scheduled ? fmt(task.scheduled) : "Schedule"}
            value={task.scheduled ? dayOf(task.scheduled) : null}
            today={todayKey()}
            late={!!task.scheduled && task.status !== "done" && isLate(task.scheduled)}
            onSelect={schedule}
          />
          {task.waiting ? <Caption>⏳ {task.waiting}</Caption> : null}
          <Stamp icon={CirclePlusIcon} label="created" value={fmtDay(task.created)} detail={fmt(task.created)} />
          <Stamp icon={PencilIcon} label="updated" value={fmtDay(task.updated)} detail={fmt(task.updated)} />
        </Group>
        {task.tickets.length + task.prs.length + task.links.length ? (
          <Group gap="md">
            {links(task.tickets, TicketIcon)}
            {links(task.prs, GitPullRequestIcon)}
            {links(task.links, LinkIcon)}
          </Group>
        ) : null}
      </Stack>
      <ErrorText>{error}</ErrorText>
      <Section>
        <Prose html={task.html} />
      </Section>
      <DropDialog id={task.id} opened={dropping} onClose={() => setDropping(false)} onDropped={onChanged} />
    </Stack>
  );
}
