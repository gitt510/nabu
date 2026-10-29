import { useState } from "react";

import { LINE_LIMIT } from "../../api/schema";
import { patch } from "../api";
import { Button, Dialog, Form, Group, LinesInput, Stack } from "./ui";

/** dropped にする前に、やらない理由を聞く。1 行 1 項目 */
export function DropDialog({ id, opened, onClose, onDropped }: { id: number; opened: boolean; onClose: () => void; onDropped: () => void }) {
  const [why, setWhy] = useState("");
  const [error, setError] = useState<string>();
  const submit = async () => {
    const lines = why.split("\n").map((l) => l.trim()).filter(Boolean);
    if (!lines.length) return setError("Write at least one reason.");
    const problem = await patch(id, { status: "dropped", why_dropped: lines });
    setError(problem);
    if (problem) return;
    setWhy("");
    onClose();
    onDropped();
  };
  return (
    <Dialog title="Drop" opened={opened} onClose={onClose}>
      <Form onSubmit={() => void submit()}>
        <Stack gap="md">
          <LinesInput label="Why drop it?" description={`One reason per line, up to ${LINE_LIMIT} characters each`} value={why} onChange={setWhy} error={error} />
          <Group justify="flex-end">
            <Button type="submit">Drop task</Button>
          </Group>
        </Stack>
      </Form>
    </Dialog>
  );
}
