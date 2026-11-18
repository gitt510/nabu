import type { TaskView } from "../api";
import { fmtDay } from "../time";
import { STATUS_LABEL } from "./states";
import { CalendarIcon, Caption, FolderIcon, Group } from "./ui";

/** 一覧の行に出す task の題。番号（#6）を薄く添え、CLI に渡す id がすぐ分かるように。project があれば後ろに添える */
export const TaskTitle = ({ task }: { task: TaskView }) => (
  <Group gap="xs" wrap="nowrap">
    <span>
      {task.title} <Caption mono>#{task.id}</Caption>
    </span>
    {task.project ? <Caption icon={FolderIcon}>{task.project}</Caption> : null}
  </Group>
);

/**
 * 一覧の行の右端。日付を 1 つ。閉じた task は status と直した日、
 * 開いている task は予定日（暦の icon つき）、予定が無ければ作った日
 */
export const TaskWhen = ({ task }: { task: TaskView }) => (
  <Group gap={6} wrap="nowrap">
    {task.status !== "todo" ? (
      <>
        <Caption>{STATUS_LABEL[task.status]}</Caption>
        <Caption mono>{fmtDay(task.updated)}</Caption>
      </>
    ) : task.scheduled ? (
      <Caption mono icon={CalendarIcon}>
        {fmtDay(task.scheduled)}
      </Caption>
    ) : (
      <Caption mono>{fmtDay(task.created)}</Caption>
    )}
  </Group>
);
