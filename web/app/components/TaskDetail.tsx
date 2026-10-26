import { useState } from "react";

import type { Status } from "../../src/fields";
import { type Item, patch } from "../api";
import { fmt, isLate, ticket } from "../time";
import { DropDialog } from "./DropDialog";
import { STATUS_CHOICES } from "./statuses";
import { BackIcon, ChoiceMenu, CopyText, ErrorText, ExternalLink, Facts, Group, Heading, IconButton, Prose, Stack, Tag } from "./ui";

/** 開いている task：slug と status、題、properties、本文。値のない property は出さない */
export function TaskDetail({ task, onBack, onChanged, menuOpened, onMenuOpenedChange }: { task: Item; onBack?: () => void; onChanged: () => void; menuOpened: boolean; onMenuOpenedChange: (o: boolean) => void }) {
  const [error, setError] = useState<string>();
  const [dropping, setDropping] = useState(false);

  const move = async (status: Status) => {
    if (status === task.status) return;
    if (status === "dropped") return setDropping(true);
    const problem = await patch(task.slug, { status });
    setError(problem);
    if (!problem) onChanged();
  };

  const links = (urls: string[]) => urls.map((u) => <ExternalLink key={u} href={u}>{ticket(u)}</ExternalLink>);
  const facts = [
    { label: "area", value: <Tag>{task.area}</Tag> },
    task.project && { label: "project", value: <Tag>{task.project}</Tag> },
    task.scheduled && { label: "scheduled", value: <Tag tone={task.status !== "done" && isLate(task.scheduled) ? "late" : "default"}>{fmt(task.scheduled)}</Tag> },
    task.waiting && { label: "waiting", value: <>⏳ {task.waiting}</> },
    { label: "created", value: fmt(task.created) },
    { label: "updated", value: fmt(task.updated) },
    task.tickets.length > 0 && { label: "tickets", value: links(task.tickets) },
    task.prs.length > 0 && { label: "prs", value: links(task.prs) },
    task.links.length > 0 && { label: "links", value: links(task.links) },
  ].filter((f) => !!f);

  return (
    <Stack gap="md">
      <Group justify="space-between" wrap="nowrap">
        <Group gap="xs" wrap="nowrap">
          {onBack ? <IconButton label="Back to list" icon={BackIcon} onClick={onBack} /> : null}
          <CopyText value={task.slug} />
        </Group>
        <ChoiceMenu choices={STATUS_CHOICES} value={task.status} onSelect={move} opened={menuOpened} onOpenedChange={onMenuOpenedChange} />
      </Group>
      <ErrorText>{error}</ErrorText>
      <Heading>{task.title}</Heading>
      <Facts items={facts} />
      <Prose html={task.html} />
      <DropDialog slug={task.slug} opened={dropping} onClose={() => setDropping(false)} onDropped={onChanged} />
    </Stack>
  );
}
