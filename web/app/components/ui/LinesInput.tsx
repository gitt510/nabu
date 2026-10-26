import { Textarea } from "@mantine/core";

/** 複数行の入力 */
export const LinesInput = ({ label, description, value, onChange, error }: { label: string; description: string; value: string; onChange: (v: string) => void; error?: string }) => (
  <Textarea label={label} description={description} value={value} onChange={(e) => onChange(e.currentTarget.value)} error={error} autosize minRows={3} data-autofocus />
);
