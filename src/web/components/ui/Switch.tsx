import { Switch as MantineSwitch } from "@mantine/core";

/** on / off の切り替え */
export const Switch = ({
  label,
  checked,
  onChange,
}: {
  label: string;
  checked: boolean;
  onChange: (v: boolean) => void;
}) => (
  <MantineSwitch
    size="sm"
    label={label}
    checked={checked}
    onChange={(e) => onChange(e.currentTarget.checked)}
  />
);
