import { Dialog, KeyTable } from "./ui";

export const KEYS = [
  { keys: ["←", "→"], label: "Home: previous / next day" },
  { keys: ["↓", "↑"], label: "Open next / previous task" },
  { keys: ["m"], label: "Add a memo line to the open task" },
  { keys: ["c"], label: "Cancel the open task, saying why" },
  { keys: ["⌘ Enter"], label: "Cancel it, once the reason is written" },
  { keys: ["?"], label: "Show this list" },
];

export const HelpDialog = ({ opened, onClose }: { opened: boolean; onClose: () => void }) => (
  <Dialog title="Keyboard" opened={opened} onClose={onClose}>
    <KeyTable rows={KEYS} />
  </Dialog>
);
