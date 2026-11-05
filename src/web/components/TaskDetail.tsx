import { useState } from "react";

import type { Task } from "../../api/schema";
import { check, type Item, patch } from "../api";
import { dayOf, fmt, fmtDay, ticket, timeOf, todayKey } from "../time";
import { DropDialog } from "./DropDialog";
import { Memo } from "./Memo";
import { MemoDialog } from "./MemoDialog";
import { STATUS_LABEL } from "./states";
import {
  ArrowBackUpIcon,
  ArrowLeftIcon,
  Caption,
  CircleCheckIcon,
  CirclePlusIcon,
  CircleXIcon,
  DatePickerButton,
  ErrorText,
  ExternalLink,
  FolderIcon,
  GitPullRequestIcon,
  Group,
  Heading,
  HourglassIcon,
  IconButton,
  type IconComponent,
  LinkIcon,
  NotesIcon,
  Pane,
  PencilIcon,
  Prose,
  Passage,
  Stack,
  Timestamp,
  TagIcon,
  TicketIcon,
  useHotkeys,
} from "./ui";

/** 開いている task：番号を添えた題と右端の memo を足す・完了・やめる（閉じた task は開き直す）操作、その下に予定日の札・作った日・直した日の 1 行と、area・project・waiting・link を icon つきで並べる 1 行、端から端までの区切りの線のあとに、scroll する本文として Why と DoD（と Why dropped）を枠なしの文章で、そのあとに書き足せる Memo を。書かれていない section は見出しだけを出す */
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
  const [noting, setNoting] = useState(false);

  const settle = (problem: string | undefined) => {
    setError(problem);
    if (!problem) onChanged();
  };
  const save = async (fields: Partial<Task>) => settle(await patch(task.id, fields));
  const close = async (done: boolean) => settle(await check(task.id, done));
  useHotkeys([
    ["d", () => task.status === "todo" && setDropping(true)],
    ["m", () => setNoting(true)],
  ]);

  const actions = (
    <Group gap={4} wrap="nowrap">
      <IconButton label="Add memo… (m)" icon={NotesIcon} onClick={() => setNoting(true)} />
      {task.status === "todo" ? (
        <>
          <IconButton label="Mark done" icon={CircleCheckIcon} onClick={() => void close(true)} />
          <IconButton label="Drop… (d)" icon={CircleXIcon} onClick={() => setDropping(true)} />
        </>
      ) : (
        <IconButton label="Reopen" icon={ArrowBackUpIcon} onClick={() => void close(false)} />
      )}
    </Group>
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
          <DatePickerButton
            label={task.scheduled ? fmt(task.scheduled) : "Schedule"}
            value={task.scheduled ? dayOf(task.scheduled) : null}
            today={todayKey()}
            onSelect={schedule}
          />
          <Timestamp
            icon={CirclePlusIcon}
            label="created"
            value={fmtDay(task.created)}
            detail={fmt(task.created)}
          />
          <Timestamp
            icon={PencilIcon}
            label="updated"
            value={fmtDay(task.updated)}
            detail={fmt(task.updated)}
          />
        </Group>
        <Group gap="md">
          <Caption icon={TagIcon}>{task.area}</Caption>
          {task.project ? <Caption icon={FolderIcon}>{task.project}</Caption> : null}
          {task.waiting ? <Caption icon={HourglassIcon}>{task.waiting}</Caption> : null}
          {links(task.tickets, TicketIcon)}
          {links(task.prs, GitPullRequestIcon)}
          {links(task.links, LinkIcon)}
        </Group>
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
        <Memo
          key={task.id}
          lines={task.memo}
          onSave={async (memo) => {
            const problem = await patch(task.id, { memo });
            if (!problem) onChanged();
            return problem;
          }}
        />
      </Stack>
      <DropDialog
        id={task.id}
        opened={dropping}
        onClose={() => setDropping(false)}
        onDropped={onChanged}
      />
      <MemoDialog
        id={task.id}
        memo={task.memo}
        opened={noting}
        onClose={() => setNoting(false)}
        onAdded={onChanged}
      />
    </Pane>
  );
}
