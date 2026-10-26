import { Button as MantineButton, Menu } from "@mantine/core";

import { CheckIcon, type IconComponent } from "./icons";

export type Choice<V extends string> = { value: V; label: string; icon: IconComponent; apart?: boolean };

/** 今の値を出す button と、選び直す menu。apart の項目は線で区切って下に置く */
export function ChoiceMenu<V extends string>({ choices, value, onSelect, opened, onOpenedChange }: { choices: Choice<V>[]; value: V; onSelect: (v: V) => void; opened: boolean; onOpenedChange: (o: boolean) => void }) {
  const current = choices.find((c) => c.value === value) ?? choices[0];
  return (
    <Menu opened={opened} onChange={onOpenedChange} position="bottom-end" width={180}>
      <Menu.Target>
        <MantineButton variant="default" size="xs" leftSection={<current.icon size={16} />}>
          {current.label}
        </MantineButton>
      </Menu.Target>
      <Menu.Dropdown>
        {choices.map((c) => [
          c.apart ? <Menu.Divider key={`${c.value}-divider`} /> : null,
          <Menu.Item key={c.value} leftSection={<c.icon size={16} />} rightSection={c.value === value ? <CheckIcon size={14} /> : null} onClick={() => onSelect(c.value)}>
            {c.label}
          </Menu.Item>,
        ])}
      </Menu.Dropdown>
    </Menu>
  );
}
