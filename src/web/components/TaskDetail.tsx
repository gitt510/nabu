import { useState } from "react";

import type { Task } from "../../api/schema";
import { check, type Item, patch } from "../api";
import { dayOf, fmt, fmtDay, ticket, timeOf, todayKey } from "../time";
import { DropDialog } from "./DropDialog";
import { STATUS_LABEL } from "./states";
import {
  ArrowBackUpIcon,
  ArrowLeftIcon,
  Caption,
  CircleCheckIcon,
  CirclePlusIcon,
  CircleXIcon,
  DateMenu,
  ErrorText,
  ExternalLink,
  GitPullRequestIcon,
  Group,
  Heading,
  IconButton,
  type IconComponent,
  LinkIcon,
  Pane,
  PencilIcon,
  Prose,
  Passage,
  Stack,
  Stamp,
  Tag,
  TicketIcon,
  useHotkeys,
} from "./ui";

/** 開いている task：番号を添えた題と右端の完了・やめる（閉じた task は開き直す）操作、その下に properties と作った日・直した日の 1 行と link の 1 行、端から端までの区切りの線のあとに、scroll する本文として Why と DoD（と Why dropped）を枠なしの文章で。書かれていない section は見出しだけを出す。予定日以外の値のない property は出さない */
export function TaskDetail({
  task,
  onBack,
  onChanged,
}: {
  task: Item;
  onBack?: () => void;
  onChanged: () => void;
}) {
  const [error, setError] = useState<string>();
  const [dropping, setDropping] = useState(false);

  const settle = (problem: string | undefined) => {
    setError(problem);
    if (!problem) onChanged();
  };
  const save = async (fields: Partial<Task>) => settle(await patch(task.id, fields));
  const close = async (done: boolean) => settle(await check(task.id, done));
  useHotkeys([["d", () => task.status === "todo" && setDropping(true)]]);

  const actions =
    task.status === "todo" ? (
      <Group gap={4} wrap="nowrap">
        <IconButton label="Mark done" icon={CircleCheckIcon} onClick={() => void close(true)} />
        <IconButton label="Drop… (d)" icon={CircleXIcon} onClick={() => setDropping(true)} />
      </Group>
    ) : (
      <IconButton label="Reopen" icon={ArrowBackUpIcon} onClick={() => void close(false)} />
    );

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

  const head = (
    <Stack gap="xs">
      {onBack ? (
        <Group>
          <IconButton label="Back to list" icon={ArrowLeftIcon} onClick={onBack} />
        </Group>
      ) : null}
      <Stack gap="xs">
        <Heading number={`#${task.id}`} aside={actions}>
          {task.title}
        </Heading>
        <Group gap="xs">
          {task.status === "todo" ? null : <Caption>{STATUS_LABEL[task.status]}</Caption>}
          <Tag>{task.area}</Tag>
          {task.project ? <Tag>{task.project}</Tag> : null}
          <DateMenu
            label={task.scheduled ? fmt(task.scheduled) : "Schedule"}
            value={task.scheduled ? dayOf(task.scheduled) : null}
            today={todayKey()}
            onSelect={schedule}
          />
          {task.waiting ? <Caption>⏳ {task.waiting}</Caption> : null}
          <Stamp
            icon={CirclePlusIcon}
            label="created"
            value={fmtDay(task.created)}
            detail={fmt(task.created)}
          />
          <Stamp
            icon={PencilIcon}
            label="updated"
            value={fmtDay(task.updated)}
            detail={fmt(task.updated)}
          />
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
    </Stack>
  );

  return (
    <Pane head={head}>
      <Stack gap="md">
        {task.parts.map((p) => (
          <Passage key={p.key} title={p.title}>
            {p.html ? <Prose html={p.html} /> : null}
          </Passage>
        ))}
      </Stack>
      <DropDialog
        id={task.id}
        opened={dropping}
        onClose={() => setDropping(false)}
        onDropped={onChanged}
      />
    </Pane>
  );
}
