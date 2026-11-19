import { Group, Radio } from "@mantine/core";

/** 少ない選択肢から 1 つ選ぶ radio。題の下に選択肢を横に並べる */
export const Choice = <T extends string>({
  label,
  data,
  value,
  onChange,
}: {
  label: string;
  data: readonly T[];
  value: T;
  onChange: (v: T) => void;
}) => (
  <Radio.Group label={label} value={value} onChange={(v) => onChange(v as T)}>
    <Group gap="md" mt={4}>
      {data.map((d) => (
        <Radio key={d} value={d} label={d} />
      ))}
    </Group>
  </Radio.Group>
);
