import { useState } from "react";

import type { Item } from "../../api/schema";
import { addItem, deleteItem, patchItem } from "../api";
import { useChecklist } from "../hooks/useChecklist";
import {
  Button,
  Caption,
  Checkbox,
  ErrorText,
  Form,
  Group,
  IconButton,
  LineInput,
  Passage,
  PencilIcon,
  Stack,
  TrashIcon,
} from "./ui";

/**
 * task の Checklist：手順の行を並べ、ここで付け外しする。足すのは下の欄から、行は鉛筆でその場で直し、ゴミ箱で消す。
 * 行を全部付けても task の status は変えない
 */
export function Checklist({ taskId }: { taskId: number }) {
  const { items, error, write } = useChecklist(taskId);
  const [text, setText] = useState("");
  // 直している行は 1 つだけ
  const [editing, setEditing] = useState<{ id: number; text: string } | null>(null);

  const add = async () => {
    if (!text.trim()) return;
    if (await write(addItem(taskId, text.trim()))) setText("");
  };
  const rename = async () => {
    if (!editing?.text.trim()) return;
    if (await write(patchItem(editing.id, { text: editing.text.trim() }))) setEditing(null);
  };

  const row = (i: Item) =>
    editing?.id === i.id ? (
      <Form key={i.id} onSubmit={() => void rename()}>
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
    ) : (
      <Group key={i.id} justify="space-between" wrap="nowrap">
        <Checkbox
          label={i.text}
          checked={i.done}
          onChange={(v) => void write(patchItem(i.id, { done: v }))}
        />
        <Group gap={4} wrap="nowrap">
          <IconButton
            label="Rename"
            icon={PencilIcon}
            onClick={() => setEditing({ id: i.id, text: i.text })}
          />
          <IconButton
            label="Delete"
            icon={TrashIcon}
            onClick={() => void write(deleteItem(i.id))}
          />
        </Group>
      </Group>
    );

  return (
    <Passage
      title="Checklist"
      aside={
        items?.length ? (
          <Caption mono>
            {items.filter((i) => i.done).length} / {items.length}
          </Caption>
        ) : null
      }
    >
      <Stack gap="xs">
        {items?.map(row)}
        <Form onSubmit={() => void add()}>
          <Group gap="xs" wrap="nowrap">
            <LineInput label="New step" placeholder="Add a step" value={text} onChange={setText} />
            <Button small type="submit">
              Add
            </Button>
          </Group>
        </Form>
        <ErrorText>{error}</ErrorText>
      </Stack>
    </Passage>
  );
}
