import { useState } from "react";

import { patch } from "../api";
import { Button, Dialog, ErrorText, Form, Group, LineInput, Stack } from "./ui";

/** Memo の末尾に 1 行足す。Enter で足して閉じる */
export function MemoDialog({
  id,
  memo,
  opened,
  onClose,
  onAdded,
}: {
  id: number;
  memo: string[];
  opened: boolean;
  onClose: () => void;
  onAdded: () => void;
}) {
  const [line, setLine] = useState("");
  const [error, setError] = useState<string>();
  const submit = async () => {
    if (!line.trim()) return;
    const problem = await patch(id, { memo: [...memo, line] });
    setError(problem);
    if (problem) return;
    setLine("");
    onClose();
    onAdded();
  };
  return (
    <Dialog title="Memo" opened={opened} onClose={onClose}>
      <Form onSubmit={() => void submit()}>
        <Stack gap="md">
          <LineInput label="New note" value={line} onChange={setLine} autoFocus />
          <ErrorText>{error}</ErrorText>
          <Group justify="flex-end">
            <Button type="submit">Add</Button>
          </Group>
        </Stack>
      </Form>
    </Dialog>
  );
}
