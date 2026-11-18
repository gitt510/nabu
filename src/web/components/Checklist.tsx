import { useState } from "react";

import type { Item } from "../../api/schema";
import { addItem, deleteItem, patchItem } from "../api";
import { useChecklist } from "../hooks/useChecklist";
import {
  Button,
  Caption,
  CheckMark,
  ErrorText,
  Form,
  Indent,
  Group,
  IconButton,
  LineInput,
  Passage,
  Stack,
  TextButton,
  TrashIcon,
} from "./ui";

/**
 * task の Checklist：手順の行を並べ、ここで付け外しする。足すのは下の欄から、行は文字を押してその場で直し（Enter か欄の外で保存、Esc でやめる。空にして保存すると消える）、ゴミ箱で消す。
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
  // 空にすると行を消し、変えなかったときは何も送らずにやめる
  const rename = async (i: Item) => {
    if (!editing) return;
    const next = editing.text.trim();
    if (next === i.text) return setEditing(null);
    if (await write(next ? patchItem(i.id, { text: next }) : deleteItem(i.id))) setEditing(null);
  };

  const row = (i: Item) => (
    <Group key={i.id} gap="sm" wrap="nowrap">
      <CheckMark
        label={i.text}
        checked={i.done}
        onChange={(v) => void write(patchItem(i.id, { done: v }))}
      />
      {editing?.id === i.id ? (
        <Form onSubmit={() => void rename(i)}>
          <LineInput
            label="Rename"
            value={editing.text}
            onChange={(t) => setEditing({ id: i.id, text: t })}
            onEscape={() => setEditing(null)}
            onBlur={() => void rename(i)}
            autoFocus
          />
        </Form>
      ) : (
        <TextButton
          label={`Rename ${i.text}`}
          onClick={() => setEditing({ id: i.id, text: i.text })}
        >
          {i.text}
        </TextButton>
      )}
      <IconButton label="Delete" icon={TrashIcon} onClick={() => void write(deleteItem(i.id))} />
    </Group>
  );

  const adder = (
    <Form onSubmit={() => void add()}>
      <Group gap="xs" wrap="nowrap">
        <LineInput label="New step" value={text} onChange={setText} />
        <Button small type="submit">
          Add
        </Button>
      </Group>
    </Form>
  );

  // 行が無いうちは見出しを立てず、題を添えた入力の 1 行だけにする
  if (items && !items.length)
    return (
      <Stack gap="xs">
        <Group gap="sm" wrap="nowrap">
          <Caption>Checklist</Caption>
          {adder}
        </Group>
        <ErrorText>{error}</ErrorText>
      </Stack>
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
        <Indent>{adder}</Indent>
        <ErrorText>{error}</ErrorText>
      </Stack>
    </Passage>
  );
}
