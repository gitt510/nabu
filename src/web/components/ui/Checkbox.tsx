import { Checkbox as MantineCheckbox } from "@mantine/core";

/** やった / まだ、を付け外しする 1 行 */
export const Checkbox = ({ label, checked, onChange }: { label: string; checked: boolean; onChange: (v: boolean) => void }) => (
  <MantineCheckbox label={label} checked={checked} onChange={(e) => onChange(e.currentTarget.checked)} />
);
