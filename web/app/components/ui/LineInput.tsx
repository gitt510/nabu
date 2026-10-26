import { TextInput } from "@mantine/core";

/** 1 行の入力。横に並んだときは、空いている幅いっぱいに伸びる */
export const LineInput = ({ label, placeholder, value, onChange }: { label: string; placeholder: string; value: string; onChange: (v: string) => void }) => (
  <TextInput flex={1} aria-label={label} placeholder={placeholder} autoComplete="off" value={value} onChange={(e) => onChange(e.currentTarget.value)} />
);
