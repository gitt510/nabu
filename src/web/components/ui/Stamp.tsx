import { Group, Text, Tooltip } from "@mantine/core";

import type { IconComponent } from "./icons";

/** 記録の時刻を icon と短い日付で出す。何の時刻かと詳しい時刻は hover の tooltip に任せる。数字は一覧の日付と同じ等幅で、icon と高さが揃う */
export const Stamp = ({ icon: Icon, label, value, detail }: { icon: IconComponent; label: string; value: string; detail: string }) => (
  <Tooltip label={`${label} ${detail}`}>
    <Group component="span" gap={4} wrap="nowrap" c="dimmed" aria-label={`${label} ${value}`}>
      <Icon size={14} />
      <Text span size="xs" ff="monospace">
        {value}
      </Text>
    </Group>
  </Tooltip>
);
