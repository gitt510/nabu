import { Select as MantineSelect } from "@mantine/core";

/** 候補から 1 つ選ぶ。空にすると絞り込まない */
export const Select = ({ label, data, value, onChange }: { label: string; data: string[]; value: string | null; onChange: (v: string | null) => void }) => (
  <MantineSelect size="xs" placeholder={label} aria-label={label} data={data} value={value} onChange={onChange} clearable searchable flex={1} />
);
