import { useState } from "react";

import {
  ErrorText,
  Form,
  Group,
  IconButton,
  LineInput,
  Passage,
  Stack,
  TextButton,
  TrashIcon,
} from "./ui";

/**
 * task の Memo：やったこと、思ったことの行。足すのは MemoDialog から。行が無ければ何も出さない。
 * 行は文字を押してその場で直し（Enter か欄の外で保存、Esc でやめる。空にして保存すると消える）、ゴミ箱で消す。onSave は memo 全体を送り、断られたら理由を返す
 */
export function Memo({
  lines,
  onSave,
}: {
  lines: string[];
  onSave: (memo: string[]) => Promise<string | undefined>;
}) {
  // 直している行は 1 つだけ
  const [editing, setEditing] = useState<{ i: number; text: string } | null>(null);
  const [error, setError] = useState<string>();

  const save = async (memo: string[]) => {
    const problem = await onSave(memo);
    setError(problem);
    return !problem;
  };
  const edit = async () => {
    if (!editing) return;
    const { i, text } = editing;
    if (text === lines[i]) return setEditing(null);
    const memo = text.trim() ? lines.with(i, text) : lines.toSpliced(i, 1);
    if (await save(memo)) setEditing(null);
  };

  const row = (line: string, i: number) => (
    <Group key={i} gap="xs" wrap="nowrap">
      {editing?.i === i ? (
        <Form onSubmit={() => void edit()}>
          <LineInput
            label="Edit note"
            value={editing.text}
            onChange={(text) => setEditing({ i, text })}
            onEscape={() => setEditing(null)}
            onBlur={() => void edit()}
            autoFocus
          />
        </Form>
      ) : (
        <TextButton label={`Edit ${line}`} onClick={() => setEditing({ i, text: line })}>
          {line}
        </TextButton>
      )}
      <IconButton
        label="Delete"
        icon={TrashIcon}
        onClick={() => void save(lines.toSpliced(i, 1))}
      />
    </Group>
  );

  // 空なら段ごと出さない。足すのは TaskDetail の menu の Add memo から
  if (!lines.length) return null;
  return (
    <Passage title="Memo">
      <Stack gap="xs">
        {lines.map(row)}
        <ErrorText>{error}</ErrorText>
      </Stack>
    </Passage>
  );
}
