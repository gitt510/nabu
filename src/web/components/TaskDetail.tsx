import { useState } from "react";

import type { Task } from "../../api/schema";
import { check, type TaskView, patch } from "../api";
import { dayOf, fmt, fmtDay, ticket, timeOf, todayKey } from "../time";
import { Checklist } from "./Checklist";
import { CancelDialog } from "./CancelDialog";
import { Memo } from "./Memo";
import { MemoDialog } from "./MemoDialog";
import { STATUS_LABEL } from "./states";
import {
  ArrowBackUpIcon,
  Caption,
  CircleCheckIcon,
  CircleXIcon,
  DatePickerButton,
  ErrorText,
  ExternalLink,
  FolderIcon,
  GitPullRequestIcon,
  Group,
  HourglassIcon,
  MoreMenu,
  type IconComponent,
  LinkIcon,
  NotesIcon,
  Prose,
  Passage,
  Stack,
  Timestamp,
  TagIcon,
  TicketIcon,
  useHotkeys,
} from "./ui";

/**
 * 一覧の行の下に開いた task。題は行が見せるので、ここは Checklist や Memo と同じ形の Details の段から始める。
 * 題の右端の三点の menu に memo を足す・完了・やめる（閉じた task は開き直す）操作、中に予定日の札・作った日・直した日の 1 行と、
 * area・project・waiting・link を icon つきで並べる 1 行、
 * やめた task なら Why canceled を枠なしの文章で、そのあとに付け外しできる Checklist と、書き足せる Memo を
 */
export function TaskDetail({ task, onChanged }: { task: TaskView; onChanged: () => void }) {
  const [error, setError] = useState<string>();
  const [canceling, setCanceling] = useState(false);
  const [noting, setNoting] = useState(false);

  const settle = (problem: string | undefined) => {
    setError(problem);
    if (!problem) onChanged();
  };
  const save = async (fields: Partial<Task>) => settle(await patch(task.id, fields));
  const close = async (done: boolean) => settle(await check(task.id, done));
  useHotkeys([
    ["c", () => task.status === "todo" && setCanceling(true)],
    ["m", () => setNoting(true)],
  ]);

  const actions = (
    <MoreMenu
      label="Task actions"
      items={[
        { label: "Add memo…", icon: NotesIcon, hotkey: "m", onClick: () => setNoting(true) },
        ...(task.status === "todo"
          ? [
              { label: "Mark done", icon: CircleCheckIcon, onClick: () => void close(true) },
              {
                label: "Cancel…",
                icon: CircleXIcon,
                hotkey: "c",
                onClick: () => setCanceling(true),
              },
            ]
          : [{ label: "Reopen", icon: ArrowBackUpIcon, onClick: () => void close(false) }]),
      ]}
    />
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

  const details = (
    <Passage title="Details" aside={actions}>
      <Stack gap="xs">
        <Group gap="xs">
          {task.status === "todo" ? null : <Caption>{STATUS_LABEL[task.status]}</Caption>}
          <DatePickerButton
            label={task.scheduled ? fmt(task.scheduled) : "Schedule"}
            value={task.scheduled ? dayOf(task.scheduled) : null}
            today={todayKey()}
            onSelect={schedule}
          />
          <Timestamp label="created" value={fmtDay(task.created)} detail={fmt(task.created)} />
          <Timestamp label="updated" value={fmtDay(task.updated)} detail={fmt(task.updated)} />
        </Group>
        <Group gap="md">
          <Caption icon={TagIcon}>{task.area}</Caption>
          {task.project ? <Caption icon={FolderIcon}>{task.project}</Caption> : null}
          {task.waiting ? <Caption icon={HourglassIcon}>{task.waiting}</Caption> : null}
          {links(task.tickets, TicketIcon)}
          {links(task.prs, GitPullRequestIcon)}
          {links(task.links, LinkIcon)}
        </Group>
        <ErrorText>{error}</ErrorText>
      </Stack>
    </Passage>
  );

  return (
    <Stack gap="md">
      {details}
      {task.whyCanceled ? (
        <Passage title="Why canceled">
          <Prose html={task.whyCanceled} />
        </Passage>
      ) : null}
      <Checklist key={`checklist-${task.id}`} taskId={task.id} />
      <Memo
        key={task.id}
        lines={task.memo}
        onSave={async (memo) => {
          const problem = await patch(task.id, { memo });
          if (!problem) onChanged();
          return problem;
        }}
      />
      <CancelDialog
        id={task.id}
        opened={canceling}
        onClose={() => setCanceling(false)}
        onCanceled={onChanged}
      />
      <MemoDialog
        id={task.id}
        memo={task.memo}
        opened={noting}
        onClose={() => setNoting(false)}
        onAdded={onChanged}
      />
    </Stack>
  );
}
