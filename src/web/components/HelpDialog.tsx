import { Dialog, KeyTable } from "./ui";

export const KEYS = [
  { keys: ["←", "→"], label: "Calendar: previous / next day" },
  { keys: ["/"], label: "Focus search" },
  { keys: ["↓", "↑"], label: "Open next / previous task" },
  { keys: ["[", "]"], label: "Todo / Done tab" },
  { keys: ["?"], label: "Show this list" },
];

export const HelpDialog = ({ opened, onClose }: { opened: boolean; onClose: () => void }) => (
  <Dialog title="Keyboard" opened={opened} onClose={onClose}>
    <KeyTable rows={KEYS} />
  </Dialog>
);
