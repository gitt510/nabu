import { Text, Tooltip } from "@mantine/core";

/** 記録の時刻を、何の時刻か（created など）と短い日付で出す。詳しい時刻は hover の tooltip に任せる。数字は一覧の日付と同じ等幅 */
export const Timestamp = ({
  label,
  value,
  detail,
}: {
  label: string;
  value: string;
  detail: string;
}) => (
  <Tooltip label={`${label} ${detail}`}>
    <Text span size="xs" c="dimmed">
      {label}{" "}
      <Text span inherit ff="monospace">
        {value}
      </Text>
    </Text>
  </Tooltip>
);
