import { useState } from "react";

import { patch } from "../api";
import { Button, Dialog, ErrorText, Form, Group, LineInput, Stack } from "./ui";

/** task の題を直す。今の題を入れた欄から始め、Enter で直して閉じる */
export function TitleDialog({
  id,
  title,
  opened,
  onClose,
  onSaved,
}: {
  id: number;
  title: string;
  opened: boolean;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [value, setValue] = useState(title);
  const [error, setError] = useState<string>();
  const submit = async () => {
    if (!value.trim()) return;
    if (value.trim() === title) return onClose();
    const problem = await patch(id, { title: value.trim() });
    setError(problem);
    if (problem) return;
    onClose();
    onSaved();
  };
  return (
    <Dialog title="Edit title" opened={opened} onClose={onClose}>
      <Form onSubmit={() => void submit()}>
        <Stack gap="md">
          <LineInput label="Title" value={value} onChange={setValue} autoFocus />
          <ErrorText>{error}</ErrorText>
          <Group justify="flex-end">
            <Button type="submit">Save</Button>
          </Group>
        </Stack>
      </Form>
    </Dialog>
  );
}
