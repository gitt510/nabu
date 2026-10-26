import type { ComponentType, ReactNode, Ref } from "react";
import { ActionIcon, Button as MantineButton, Chip, CopyButton, Group, Menu, NavLink, Select as MantineSelect, Tabs, Text, Textarea, TextInput, Title, Tooltip, UnstyledButton } from "@mantine/core";

import { CheckIcon, NextIcon, PrevIcon, SearchIcon } from "./icons";

type Icon = ComponentType<{ size?: number; stroke?: number }>;

/** 一覧の上の検索欄 */
export const SearchInput = ({ value, onChange, ref }: { value: string; onChange: (v: string) => void; ref?: Ref<HTMLInputElement> }) => (
  <TextInput
    ref={ref}
    type="search"
    size="sm"
    placeholder="Search"
    aria-label="Search"
    autoComplete="off"
    leftSection={<SearchIcon size={16} />}
    value={value}
    onChange={(e) => onChange(e.currentTarget.value)}
  />
);

/** 1 つだけ選ぶ横並びの切り替え（status など）。中身の panel は持たず、選んだ値だけを返す */
export const Segments = <V extends string>({ items, value, onChange }: { items: { value: V; label: string }[]; value: V; onChange: (v: V) => void }) => (
  <Tabs value={value} onChange={(v) => v && onChange(v as V)}>
    <Tabs.List grow>
      {items.map((i) => (
        <Tabs.Tab key={i.value} value={i.value} px="xs">
          {i.label}
        </Tabs.Tab>
      ))}
    </Tabs.List>
  </Tabs>
);

/** 候補から 1 つ選ぶ。空にすると絞り込まない */
export const Select = ({ label, data, value, onChange }: { label: string; data: string[]; value: string | null; onChange: (v: string | null) => void }) => (
  <MantineSelect size="xs" placeholder={label} aria-label={label} data={data} value={value} onChange={onChange} clearable searchable flex={1} />
);

/** on / off の絞り込み */
export const Toggle = ({ label, icon: Icon, checked, onChange }: { label: string; icon: Icon; checked: boolean; onChange: (v: boolean) => void }) => (
  <Chip size="xs" variant="outline" checked={checked} onChange={onChange} icon={<Icon size={12} />}>
    {label}
  </Chip>
);

/** 一覧の 1 行。題と、右端の補足 */
export const ListRow = ({ id, label, aside, active, onClick }: { id: string; label: ReactNode; aside: ReactNode; active: boolean; onClick: () => void }) => (
  <NavLink id={id} label={label} rightSection={aside} active={active} onClick={onClick} noWrap variant="light" />
);

/** icon だけの button。説明は tooltip に出す */
export const IconButton = ({ label, icon: Icon, onClick }: { label: string; icon: Icon; onClick: () => void }) => (
  <Tooltip label={label}>
    <ActionIcon variant="subtle" color="gray" aria-label={label} onClick={onClick}>
      <Icon size={18} />
    </ActionIcon>
  </Tooltip>
);

/** 操作の button */
export const Button = ({ children, type = "button", onClick }: { children: ReactNode; type?: "button" | "submit"; onClick?: () => void }) => (
  <MantineButton type={type} onClick={onClick}>
    {children}
  </MantineButton>
);

/** 複数行の入力 */
export const LinesInput = ({ label, description, value, onChange, error }: { label: string; description: string; value: string; onChange: (v: string) => void; error?: string }) => (
  <Textarea label={label} description={description} value={value} onChange={(e) => onChange(e.currentTarget.value)} error={error} autosize minRows={3} data-autofocus />
);

/** click で値を copy する文字列（slug など） */
export const CopyText = ({ value }: { value: string }) => (
  <CopyButton value={value}>
    {({ copied, copy }) => (
      <Tooltip label={copied ? "Copied" : "Click to copy"}>
        <UnstyledButton onClick={copy}>
          <Text span size="sm" c="dimmed" ff="monospace">
            {value}
          </Text>
        </UnstyledButton>
      </Tooltip>
    )}
  </CopyButton>
);

export type Choice<V extends string> = { value: V; label: string; icon: Icon; apart?: boolean };

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

/** 題と、前 / 次へ送る button。reset があれば、起点（今日など）へ戻る button も並べる */
export const Pager = ({ label, prev, next, reset }: { label: ReactNode; prev: { label: string; onClick: () => void }; next: { label: string; onClick: () => void }; reset?: { label: string; onClick: () => void } }) => (
  <Group justify="space-between" wrap="nowrap">
    <Title order={1} size="h2" lh={1.35}>
      {label}
    </Title>
    <Group gap="xs" wrap="nowrap">
      {reset ? (
        <MantineButton variant="default" size="xs" onClick={reset.onClick}>
          {reset.label}
        </MantineButton>
      ) : null}
      {[{ ...prev, icon: PrevIcon }, { ...next, icon: NextIcon }].map((b) => (
        <Tooltip key={b.label} label={b.label}>
          <ActionIcon variant="default" size="lg" aria-label={b.label} onClick={b.onClick}>
            <b.icon size={18} />
          </ActionIcon>
        </Tooltip>
      ))}
    </Group>
  </Group>
);
