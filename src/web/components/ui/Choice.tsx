import { Group, Radio } from "@mantine/core";

/** 少ない選択肢から 1 つ選ぶ radio。横に並べる */
export const Choice = ({
  label,
  data,
  value,
  onChange,
}: {
  label: string;
  data: { value: string; label: string }[];
  value: string;
  onChange: (v: string) => void;
}) => (
  <Radio.Group label={label} value={value} onChange={onChange} size="xs">
    <Group gap="md" mt={6}>
      {data.map((d) => (
        <Radio key={d.value} value={d.value} label={d.label} size="xs" />
      ))}
    </Group>
  </Radio.Group>
);
