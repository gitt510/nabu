import { useState } from "react";

import type { Status } from "../../api/schema";
import { type Item, patch } from "../api";
import { fmt, isLate, ticket } from "../time";
import { DropDialog } from "./DropDialog";
import { STATUS_CHOICES } from "./statuses";
import { ArrowLeftIcon, Caption, ChoiceMenu, CopyText, ErrorText, ExternalLink, Group, Heading, IconButton, Prose, Section, Stack, Tag } from "./ui";

/** 開いている task：番号を添えた題と右端の status、その下に properties の 1 行、枠の中に本文。値のない property は出さない */
export function TaskDetail({ task, onBack, onChanged, menuOpened, onMenuOpenedChange }: { task: Item; onBack?: () => void; onChanged: () => void; menuOpened: boolean; onMenuOpenedChange: (o: boolean) => void }) {
  const [error, setError] = useState<string>();
  const [dropping, setDropping] = useState(false);

  const move = async (status: Status) => {
    if (status === task.status) return;
    if (status === "dropped") return setDropping(true);
    const problem = await patch(task.id, { status });
    setError(problem);
    if (!problem) onChanged();
  };

  const links = (urls: string[]) => urls.map((u) => <ExternalLink key={u} href={u}>{ticket(u)}</ExternalLink>);

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
          {task.scheduled ? <Tag tone={task.status !== "done" && isLate(task.scheduled) ? "late" : "default"}>{fmt(task.scheduled)}</Tag> : null}
          {task.waiting ? <Caption>⏳ {task.waiting}</Caption> : null}
          {links([...task.tickets, ...task.prs, ...task.links])}
          <Caption>
            created {fmt(task.created)} · updated {fmt(task.updated)}
          </Caption>
        </Group>
      </Stack>
      <ErrorText>{error}</ErrorText>
      <Section>
        <Prose html={task.html} />
      </Section>
      <DropDialog id={task.id} opened={dropping} onClose={() => setDropping(false)} onDropped={onChanged} />
    </Stack>
  );
}
