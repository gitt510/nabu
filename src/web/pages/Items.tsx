import { useState } from "react";

import type { Item } from "../../api/schema";
import { addItem, deleteItem, patchItem } from "../api";
import { ItemTree } from "../components/ItemTree";
import {
  Button,
  Caption,
  Checkbox,
  DatePickerButton,
  EmptyState,
  ErrorText,
  Form,
  Group,
  IconButton,
  LineInput,
  Page,
  PencilIcon,
  PlusIcon,
  Section,
  Stack,
  TrashIcon,
} from "../components/ui";
import { useItems } from "../hooks/useItems";
import { childrenOf, isDone, leavesUnder } from "../outline";
import { fmtDay, todayKey } from "../time";

/**
 * 箇条書きの木を書く page。上の欄で root を足し、日を付けると Calendar のその月に出る。行の + で下に行を足し、
 * 鉛筆で直し、ゴミ箱で下の行ごと消す（戻せないので Delete をもう 1 度押して確かめる）。
 * 子のある行の印は子から決まり、付け外しすると下の行を全部付け外す
 */
export function Items() {
  const { items, error, write } = useItems();
  const [text, setText] = useState("");
  const [due, setDue] = useState<string | null>(null);
  // 1 度に触れる行は 1 つ：下に行を足している行、文を直している行、消すのを確かめている行
  const [adding, setAdding] = useState<{ parent: number; text: string } | null>(null);
  const [editing, setEditing] = useState<{ id: number; text: string } | null>(null);
  const [deleting, setDeleting] = useState<number | null>(null);
  const focus = (next: "adding" | "editing" | "deleting") => {
    if (next !== "adding") setAdding(null);
    if (next !== "editing") setEditing(null);
    if (next !== "deleting") setDeleting(null);
  };

  const addRoot = async () => {
    if (!text.trim()) return;
    if (await write(addItem(text.trim(), null, due))) {
      setText("");
      setDue(null);
    }
  };
  const addChild = async () => {
    if (!adding?.text.trim()) return;
    if (await write(addItem(adding.text.trim(), adding.parent))) setAdding(null);
  };
  const rename = async () => {
    if (!editing?.text.trim()) return;
    if (await write(patchItem(editing.id, { text: editing.text.trim() }))) setEditing(null);
  };

  const row = (all: Item[]) => (i: Item) => {
    if (editing?.id === i.id)
      return (
        <Form onSubmit={() => void rename()}>
          <Group gap="xs" wrap="nowrap">
            <LineInput
              label="Rename"
              value={editing.text}
              onChange={(t) => setEditing({ id: i.id, text: t })}
              onEscape={() => setEditing(null)}
              autoFocus
            />
            <Button small onClick={() => setEditing(null)}>
              Cancel
            </Button>
            <Button small type="submit">
              Save
            </Button>
          </Group>
        </Form>
      );
    const done = isDone(all, i);
    return (
      <Stack gap={4}>
        <Group justify="space-between" wrap="nowrap">
          <Checkbox
            label={i.text}
            checked={done}
            onChange={(v) =>
              void write(...leavesUnder(all, i).map((l) => patchItem(l.id, { done: v })))
            }
          />
          {deleting === i.id ? (
            <Group gap="xs" wrap="nowrap">
              <Button small onClick={() => setDeleting(null)}>
                Cancel
              </Button>
              <Button small onClick={() => void write(deleteItem(i.id))}>
                Delete
              </Button>
            </Group>
          ) : (
            <Group gap={4} wrap="nowrap">
              {i.due ? <Caption mono>{fmtDay(i.due)}</Caption> : null}
              <IconButton
                label="Add a line under it"
                icon={PlusIcon}
                onClick={() => {
                  focus("adding");
                  setAdding({ parent: i.id, text: "" });
                }}
              />
              <IconButton
                label="Rename"
                icon={PencilIcon}
                onClick={() => {
                  focus("editing");
                  setEditing({ id: i.id, text: i.text });
                }}
              />
              <IconButton
                label="Delete with the lines under it"
                icon={TrashIcon}
                onClick={() => {
                  focus("deleting");
                  setDeleting(i.id);
                }}
              />
            </Group>
          )}
        </Group>
        {adding?.parent === i.id ? (
          <Form onSubmit={() => void addChild()}>
            <Group gap="xs" wrap="nowrap">
              <LineInput
                label="New line"
                placeholder={`Under ${i.text}`}
                value={adding.text}
                onChange={(t) => setAdding({ parent: i.id, text: t })}
                onEscape={() => setAdding(null)}
                autoFocus
              />
              <Button small type="submit">
                Add
              </Button>
            </Group>
          </Form>
        ) : null}
      </Stack>
    );
  };

  const roots = items ? childrenOf(items, null) : [];
  return (
    <Page width="narrow">
      <Section title="Items" aside={items ? <Caption mono>{roots.length}</Caption> : null}>
        <Stack gap="sm">
          <Form onSubmit={() => void addRoot()}>
            <Group gap="xs" wrap="nowrap">
              <LineInput
                label="New item"
                placeholder="Something to get done, with lines under it"
                value={text}
                onChange={setText}
              />
              <DatePickerButton
                label={due ? fmtDay(due) : "Due"}
                value={due}
                today={todayKey()}
                onSelect={setDue}
              />
              <Button type="submit">Add</Button>
            </Group>
          </Form>
          <ErrorText>{error}</ErrorText>
          {!items ? (
            <EmptyState title="Loading…" />
          ) : roots.length ? (
            <ItemTree items={items} roots={roots} row={row(items)} />
          ) : (
            <EmptyState title="No items yet" description="Add one above." />
          )}
        </Stack>
      </Section>
    </Page>
  );
}
