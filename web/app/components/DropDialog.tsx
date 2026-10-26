import { useState } from "react";

import { LINE_LIMIT } from "../../src/fields";
import { patch } from "../api";
import { Box, Button, Dialog, Group, LinesInput } from "./ui";

/** dropped にする前に、やらない理由を聞く。1 行 1 項目 */
export function DropDialog({ slug, opened, onClose, onDropped }: { slug: string; opened: boolean; onClose: () => void; onDropped: () => void }) {
  const [why, setWhy] = useState("");
  const [error, setError] = useState<string>();
  const submit = async () => {
    const lines = why.split("\n").map((l) => l.trim()).filter(Boolean);
    if (!lines.length) return setError("Write at least one reason.");
    const problem = await patch(slug, { status: "dropped", why_dropped: lines });
    setError(problem);
    if (problem) return;
    setWhy("");
    onClose();
    onDropped();
  };
  return (
    <Dialog title="Drop" opened={opened} onClose={onClose}>
      <Box
        component="form"
        onSubmit={(e) => {
          e.preventDefault();
          void submit();
        }}
      >
        <LinesInput label="Why drop it?" description={`One reason per line, up to ${LINE_LIMIT} characters each`} value={why} onChange={setWhy} error={error} />
        <Group justify="flex-end" mt="md">
          <Button type="submit">Drop task</Button>
        </Group>
      </Box>
    </Dialog>
  );
}
