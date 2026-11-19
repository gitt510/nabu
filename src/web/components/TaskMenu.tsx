import { useState } from "react";

import { check, type TaskView } from "../api";
import { CancelDialog } from "./CancelDialog";
import { MemoDialog } from "./MemoDialog";
import { TitleDialog } from "./TitleDialog";
import {
  ArrowBackUpIcon,
  CircleCheckIcon,
  CircleXIcon,
  ErrorText,
  Group,
  MoreMenu,
  NotesIcon,
  PencilIcon,
  useHotkeys,
} from "./ui";

/**
 * 開いている task の行の右端に置く三点の menu。題を直す・memo を足す・完了・やめる（閉じた task は開き直す）。
 * 開いている行にだけ置くので、key（e・m・c）はその task に効く
 */
export function TaskMenu({ task, onChanged }: { task: TaskView; onChanged: () => void }) {
  const [error, setError] = useState<string>();
  const [titling, setTitling] = useState(false);
  const [noting, setNoting] = useState(false);
  const [canceling, setCanceling] = useState(false);

  const close = async (done: boolean) => {
    const problem = await check(task.id, done);
    setError(problem);
    if (!problem) onChanged();
  };
  useHotkeys([
    ["e", () => setTitling(true)],
    ["m", () => setNoting(true)],
    ["c", () => task.status === "todo" && setCanceling(true)],
  ]);

  return (
    <Group gap="xs" wrap="nowrap" justify="flex-end">
      <ErrorText>{error}</ErrorText>
      <MoreMenu
        label="Task actions"
        items={[
          { label: "Edit title…", icon: PencilIcon, hotkey: "e", onClick: () => setTitling(true) },
          { label: "Add memo…", icon: NotesIcon, hotkey: "m", onClick: () => setNoting(true) },
          ...(task.status === "todo"
            ? [
                { label: "Mark done", icon: CircleCheckIcon, onClick: () => void close(true) },
                {
                  label: "Cancel…",
                  icon: CircleXIcon,
                  hotkey: "c",
                  onClick: () => setCanceling(true),
                },
              ]
            : [{ label: "Reopen", icon: ArrowBackUpIcon, onClick: () => void close(false) }]),
        ]}
      />
      {/* 開くたびに今の題から始める */}
      {titling ? (
        <TitleDialog
          id={task.id}
          title={task.title}
          opened
          onClose={() => setTitling(false)}
          onSaved={onChanged}
        />
      ) : null}
      <MemoDialog
        id={task.id}
        memo={task.memo}
        opened={noting}
        onClose={() => setNoting(false)}
        onAdded={onChanged}
      />
      <CancelDialog
        id={task.id}
        opened={canceling}
        onClose={() => setCanceling(false)}
        onCanceled={onChanged}
      />
    </Group>
  );
}
