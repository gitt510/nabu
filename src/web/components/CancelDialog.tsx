import { useState } from "react";

import { LINE_LIMIT } from "../../api/schema";
import { patch } from "../api";
import { Button, Dialog, Form, Group, LinesInput, Stack } from "./ui";

/** やめる前に、その理由を聞く。1 行 1 項目 */
export function CancelDialog({
  id,
  opened,
  onClose,
  onCanceled,
}: {
  id: number;
  opened: boolean;
  onClose: () => void;
  onCanceled: () => void;
}) {
  const [why, setWhy] = useState("");
  const [error, setError] = useState<string>();
  const submit = async () => {
    const lines = why
      .split("\n")
      .map((l) => l.trim())
      .filter(Boolean);
    if (!lines.length) return setError("Write at least one reason.");
    const problem = await patch(id, { status: "canceled", why_canceled: lines });
    setError(problem);
    if (problem) return;
    setWhy("");
    onClose();
    onCanceled();
  };
  return (
    <Dialog title="Cancel task" opened={opened} onClose={onClose}>
      <Form onSubmit={() => void submit()}>
        <Stack gap="md">
          <LinesInput
            label="Why cancel it?"
            description={`One reason per line, up to ${LINE_LIMIT} characters each`}
            value={why}
            onChange={setWhy}
            error={error}
          />
          <Group justify="flex-end">
            <Button type="submit">Cancel task</Button>
          </Group>
        </Stack>
      </Form>
    </Dialog>
  );
}
