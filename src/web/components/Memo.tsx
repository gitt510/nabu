import { useState } from "react";

import {
  Button,
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
 * task の Memo：やったこと、思ったことを 1 行ずつ書き足す。足すのは下の欄から Enter で。
 * 行は鉛筆からその場で直し（空にして保存すると消える）、ゴミ箱で消す。onSave は memo 全体を送り、断られたら理由を返す
 */
export function Memo({
  lines,
  onSave,
}: {
  lines: string[];
  onSave: (memo: string[]) => Promise<string | undefined>;
}) {
  const [draft, setDraft] = useState("");
  // 直している行は 1 つだけ
  const [editing, setEditing] = useState<{ i: number; text: string } | null>(null);
  const [error, setError] = useState<string>();

  const save = async (memo: string[]) => {
    const problem = await onSave(memo);
    setError(problem);
    return !problem;
  };
  const add = async () => {
    if (!draft.trim()) return;
    if (await save([...lines, draft])) setDraft("");
  };
  const edit = async () => {
    if (!editing) return;
    const { i, text } = editing;
    const memo = text.trim() ? lines.with(i, text) : lines.toSpliced(i, 1);
    if (await save(memo)) setEditing(null);
  };

  const row = (line: string, i: number) =>
    editing?.i === i ? (
      <Form key={i} onSubmit={() => void edit()}>
        <Group gap="xs" wrap="nowrap">
          <LineInput
            label="Edit note"
            value={editing.text}
            onChange={(text) => setEditing({ i, text })}
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
      <Group key={i} justify="space-between" wrap="nowrap">
        {line}
        <Group gap={4} wrap="nowrap">
          <IconButton
            label="Edit"
            icon={PencilIcon}
            onClick={() => setEditing({ i, text: line })}
          />
          <IconButton
            label="Delete"
            icon={TrashIcon}
            onClick={() => void save(lines.toSpliced(i, 1))}
          />
        </Group>
      </Group>
    );

  return (
    <Passage title="Memo">
      <Stack gap="xs">
        {lines.map(row)}
        <Form onSubmit={() => void add()}>
          <LineInput
            label="New note"
            placeholder="What you did or thought"
            value={draft}
            onChange={setDraft}
          />
        </Form>
        <ErrorText>{error}</ErrorText>
      </Stack>
    </Passage>
  );
}
