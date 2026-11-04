import { Select as MantineSelect } from "@mantine/core";

/** 候補から 1 つ選ぶ。空にすると絞り込まない。小窓の中でも閉じないよう、候補はその場に開く */
export const Select = ({
  label,
  data,
  value,
  onChange,
}: {
  label: string;
  data: string[];
  value: string | null;
  onChange: (v: string | null) => void;
}) => (
  <MantineSelect
    size="xs"
    label={label}
    placeholder="All"
    data={data}
    value={value}
    onChange={onChange}
    clearable
    searchable
    comboboxProps={{ withinPortal: false }}
  />
);
