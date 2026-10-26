import { Dialog, KeyTable } from "./ui";

export const KEYS = [
  { keys: ["←", "→"], label: "Calendar: previous / next day" },
  { keys: ["0"], label: "Calendar: back to today" },
  { keys: ["/"], label: "Focus search" },
  { keys: ["j", "k"], label: "Open next / previous task" },
  { keys: ["[", "]"], label: "Previous / next status tab" },
  { keys: ["s"], label: "Sort by created: newest / oldest first" },
  { keys: ["m"], label: "Change the open task's status" },
  { keys: ["t"], label: "Switch light / dark theme" },
  { keys: ["?"], label: "Show this list" },
];

export const HelpDialog = ({ opened, onClose }: { opened: boolean; onClose: () => void }) => (
  <Dialog title="Keyboard" opened={opened} onClose={onClose}>
    <KeyTable rows={KEYS} />
  </Dialog>
);
