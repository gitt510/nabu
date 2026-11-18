import { Checkbox as MantineCheckbox } from "@mantine/core";

/** やった / まだ、の印だけ。文字は隣に別に置く（押して直せるように）。label は読み上げ用 */
export const CheckMark = ({
  label,
  checked,
  onChange,
}: {
  label: string;
  checked: boolean;
  onChange: (v: boolean) => void;
}) => (
  <MantineCheckbox
    aria-label={label}
    checked={checked}
    onChange={(e) => onChange(e.currentTarget.checked)}
  />
);
