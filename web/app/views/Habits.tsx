import { useState } from "react";

import { addHabit, deleteHabit } from "../api";
import { Button, Caption, Empty, ErrorText, Form, Group, IconButton, LineInput, Page, Section, Stack, TrashIcon } from "../components/ui";
import { useHabits } from "../hooks/useHabits";
import { todayKey } from "../time";

/**
 * 毎日やることを決める page。足すのは上の欄から。消すのは行の右からで、checks も一緒に消えて戻せないので、
 * その行で Delete をもう 1 度押して確かめる。やった印は Calendar で付ける
 */
export function Habits() {
  const { habits, error, write } = useHabits(todayKey());
  const [title, setTitle] = useState("");
  const [deleting, setDeleting] = useState<number | null>(null);
  const add = async () => {
    if (!title.trim()) return;
    if (await write(addHabit(title.trim()))) setTitle("");
  };
  const remove = async (id: number) => {
    setDeleting(null);
    await write(deleteHabit(id));
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
          {!habits ? (
            <Empty title="Loading…" />
          ) : habits.length ? (
            <Stack gap="xs">
              {habits.map((h) => (
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
                    <IconButton label={`Delete ${h.title}`} icon={TrashIcon} onClick={() => setDeleting(h.id)} />
                  )}
                </Group>
              ))}
            </Stack>
          ) : (
            <Empty title="No habits yet" description="Add one above." />
          )}
        </Stack>
      </Section>
    </Page>
  );
}
