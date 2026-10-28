import { useState } from "react";

import type { Habit } from "../../api/schema";
import { addHabit, deleteHabit, renameHabit } from "../api";
import { Button, Caption, Empty, ErrorText, Form, Group, IconButton, LineInput, Page, PencilIcon, Section, Stack, TrashIcon } from "../components/ui";
import { useHabits } from "../hooks/useHabits";
import { todayKey } from "../time";

/**
 * 毎日やることを決める page。足すのは上の欄から。題は行の鉛筆からその場で直し、checks はそのまま残る。
 * 消すのは行のゴミ箱からで、checks も一緒に消えて戻せないので、その行で Delete をもう 1 度押して確かめる。
 * やった印は Calendar で付ける
 */
export function Habits() {
  const { habits, error, write } = useHabits(todayKey());
  const [title, setTitle] = useState("");
  // 1 度に触れる行は 1 つ：題を直している行か、消すのを確かめている行
  const [editing, setEditing] = useState<{ id: number; title: string } | null>(null);
  const [deleting, setDeleting] = useState<number | null>(null);
  const add = async () => {
    if (!title.trim()) return;
    if (await write(addHabit(title.trim()))) setTitle("");
  };
  const rename = async () => {
    if (!editing?.title.trim()) return;
    if (await write(renameHabit(editing.id, editing.title.trim()))) setEditing(null);
  };
  const remove = async (id: number) => {
    setDeleting(null);
    await write(deleteHabit(id));
  };

  const row = (h: Habit) => {
    if (editing?.id === h.id) {
      return (
        <Form key={h.id} onSubmit={() => void rename()}>
          <Group gap="xs" wrap="nowrap">
            <LineInput label={`Rename ${h.title}`} value={editing.title} onChange={(t) => setEditing({ id: h.id, title: t })} onEscape={() => setEditing(null)} autoFocus />
            <Button small tone="quiet" onClick={() => setEditing(null)}>
              Cancel
            </Button>
            <Button small type="submit">
              Save
            </Button>
          </Group>
        </Form>
      );
    }
    return (
      <Group key={h.id} justify="space-between" wrap="nowrap">
        {h.title}
        {deleting === h.id ? (
          <Group gap="xs" wrap="nowrap">
            <Button small tone="quiet" onClick={() => setDeleting(null)}>
              Cancel
            </Button>
            <Button small tone="danger" onClick={() => void remove(h.id)}>
              Delete
            </Button>
          </Group>
        ) : (
          <Group gap={4} wrap="nowrap">
            <IconButton label={`Rename ${h.title}`} icon={PencilIcon} onClick={() => { setDeleting(null); setEditing({ id: h.id, title: h.title }); }} />
            <IconButton label={`Delete ${h.title}`} icon={TrashIcon} onClick={() => { setEditing(null); setDeleting(h.id); }} />
          </Group>
        )}
      </Group>
    );
  };

  return (
    <Page width="narrow">
      <Section title="Habits" aside={habits ? <Caption mono>{habits.length}</Caption> : null}>
        <Stack gap="sm">
          <Form onSubmit={() => void add()}>
            <Group gap="xs" wrap="nowrap">
              <LineInput label="New habit" placeholder="Something to do every day" value={title} onChange={setTitle} />
              <Button type="submit">Add</Button>
            </Group>
          </Form>
          <ErrorText>{error}</ErrorText>
          {!habits ? <Empty title="Loading…" /> : habits.length ? <Stack gap="xs">{habits.map(row)}</Stack> : <Empty title="No habits yet" description="Add one above." />}
        </Stack>
      </Section>
    </Page>
  );
}
