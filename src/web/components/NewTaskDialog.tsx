import { useState } from "react";

import { AREAS, LINE_LIMIT } from "../../api/schema";
import { addItem, addTask, patch } from "../api";
import { fmt, todayKey } from "../time";
import {
  Button,
  Choice,
  DatePickerButton,
  Dialog,
  ErrorText,
  FoldSections,
  Form,
  Group,
  LineInput,
  LinesInput,
  Stack,
} from "./ui";

type Area = (typeof AREAS)[number];

const linesOf = (text: string) =>
  text
    .split("\n")
    .map((l) => l.trim())
    .filter(Boolean);
const count = (n: number) => (n ? `${n} line${n === 1 ? "" : "s"}` : undefined);

/**
 * task を足す。題の下に、開いた task と同じ Details・Checklist・Memo の段を区切って並べ、どれも最初は畳んでおく。
 * 題だけ書いて Enter で足せる。畳んだ段には入れたものの要約を添える。
 * 作ってから Checklist と Memo を書き足すので、途中で断られたら、その時点までの task は残る
 */
export function NewTaskDialog({
  opened,
  onClose,
  onAdded,
}: {
  opened: boolean;
  onClose: () => void;
  onAdded: () => void;
}) {
  const [title, setTitle] = useState("");
  const [area, setArea] = useState<Area>("personal");
  const [project, setProject] = useState("");
  const [scheduled, setScheduled] = useState<string | null>(null);
  const [checklist, setChecklist] = useState("");
  const [memo, setMemo] = useState("");
  const [error, setError] = useState<string>();

  const reset = () => {
    setTitle("");
    setProject("");
    setScheduled(null);
    setChecklist("");
    setMemo("");
    setError(undefined);
  };

  const submit = async () => {
    if (!title.trim()) return;
    const made = await addTask({
      title: title.trim(),
      area,
      project: project.trim() || undefined,
      scheduled: scheduled ?? undefined,
    });
    if (typeof made === "string") return setError(made);
    const lines = linesOf(memo);
    let problem = lines.length ? await patch(made.id, { memo: lines }) : undefined;
    for (const text of linesOf(checklist)) problem ??= await addItem(made.id, text);
    onAdded();
    reset();
    // task はもうあるので、欄は空けて、もう一度送って二重に作らないようにする
    if (problem) return setError(`#${made.id} was added, but: ${problem}`);
    onClose();
  };

  return (
    <Dialog title="New task" opened={opened} onClose={onClose}>
      <Form onSubmit={() => void submit()}>
        <Stack gap="md">
          <LineInput
            label="Title"
            placeholder="Title"
            value={title}
            onChange={setTitle}
            autoFocus
          />
          <FoldSections
            items={[
              {
                value: "details",
                title: "Details",
                hint: [scheduled && fmt(scheduled), area, project.trim()]
                  .filter(Boolean)
                  .join(" · "),
                children: (
                  <Stack gap="sm">
                    <Group>
                      <DatePickerButton
                        label={scheduled ? fmt(scheduled) : "Schedule"}
                        value={scheduled}
                        today={todayKey()}
                        onSelect={setScheduled}
                      />
                    </Group>
                    <Choice label="Area" data={AREAS} value={area} onChange={setArea} />
                    <LineInput
                      label="Project"
                      placeholder="Project (kebab-case)"
                      value={project}
                      onChange={setProject}
                    />
                  </Stack>
                ),
              },
              {
                value: "checklist",
                title: "Checklist",
                hint: count(linesOf(checklist).length),
                children: (
                  <LinesInput
                    label="Checklist"
                    hideLabel
                    description="One item per line"
                    value={checklist}
                    onChange={setChecklist}
                  />
                ),
              },
              {
                value: "memo",
                title: "Memo",
                hint: count(linesOf(memo).length),
                children: (
                  <LinesInput
                    label="Memo"
                    hideLabel
                    description={`One note per line, up to ${LINE_LIMIT} characters each`}
                    value={memo}
                    onChange={setMemo}
                  />
                ),
              },
            ]}
          />
          <ErrorText>{error}</ErrorText>
          <Group justify="flex-end">
            <Button type="submit">Add</Button>
          </Group>
        </Stack>
      </Form>
    </Dialog>
  );
}
