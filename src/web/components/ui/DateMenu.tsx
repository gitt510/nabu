import { useState } from "react";
import { Button, Group, Popover } from "@mantine/core";
import { DatePicker } from "@mantine/dates";

import { CalendarIcon } from "./icons";

/** 日を選ぶ札。押すと月の暦が開き、日を選んだ時点で決まって閉じる。日は YYYY-MM-DD、外すと null */
export function DateMenu({ label, value, today, late, onSelect }: { label: string; value: string | null; today: string; late?: boolean; onSelect: (day: string | null) => void }) {
  const [opened, setOpened] = useState(false);
  const pick = (day: string | null) => {
    setOpened(false);
    onSelect(day);
  };
  return (
    <Popover opened={opened} onChange={setOpened} position="bottom-start" shadow="md" trapFocus>
      <Popover.Target>
        <Button size="compact-xs" radius="sm" variant={value ? "light" : "default"} color={late ? "red" : undefined} leftSection={<CalendarIcon size={14} />} onClick={() => setOpened((o) => !o)}>
          {label}
        </Button>
      </Popover.Target>
      <Popover.Dropdown>
        <DatePicker value={value} onChange={pick} defaultDate={value ?? today} firstDayOfWeek={1} highlightToday />
        <Group justify="space-between" mt="xs">
          <Button variant="subtle" size="compact-sm" onClick={() => pick(today)}>
            Today
          </Button>
          <Button variant="subtle" color="red" size="compact-sm" disabled={!value} onClick={() => pick(null)}>
            Clear
          </Button>
        </Group>
      </Popover.Dropdown>
    </Popover>
  );
}
