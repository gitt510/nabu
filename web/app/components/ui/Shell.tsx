import type { ComponentType, ReactNode } from "react";
import { ActionIcon, AppShell, Box, Burger, Group, Stack, Text, Tooltip, UnstyledButton } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";

import logo from "./logo.svg?raw";

type Icon = ComponentType<{ size?: number; stroke?: number }>;

export type RailItem<K extends string> = { key: K; label: string; icon: Icon; disabled?: boolean };
export type RailAction = { label: string; icon: Icon; onClick: () => void };

/**
 * 画面の枠：上の帯、左の rail、本文。rail は page の切り替え（上）と、page をまたぐ操作（下）を持つ。
 * 幅が狭いと rail は隠れ、上の帯の Burger で開く。logo は origin への link で、click すると onHome を呼ぶ
 * （新しい tab で開く操作はそのまま効く）
 */
export function Shell<K extends string>({
  items,
  current,
  onSelect,
  actions,
  onHome,
  children,
}: {
  items: RailItem<K>[];
  current: K;
  onSelect: (key: K) => void;
  actions: RailAction[];
  onHome: () => void;
  children: ReactNode;
}) {
  const [opened, { toggle, close }] = useDisclosure();
  const button = (label: string, Icon: Icon, onClick: () => void, active = false, disabled = false) => (
    <Tooltip key={label} label={disabled ? `${label} (coming soon)` : label} position="right">
      <ActionIcon
        size="xl"
        variant={active ? "light" : "subtle"}
        color={active ? undefined : "gray"}
        aria-label={label}
        aria-current={active ? "page" : undefined}
        disabled={disabled}
        onClick={() => {
          onClick();
          close();
        }}
      >
        <Icon size={22} stroke={1.75} />
      </ActionIcon>
    </Tooltip>
  );
  return (
    <AppShell header={{ height: 48 }} navbar={{ width: 64, breakpoint: "sm", collapsed: { mobile: !opened } }}>
      <AppShell.Header>
        <Group h="100%" px="md" gap="sm">
          <Burger opened={opened} onClick={toggle} hiddenFrom="sm" size="sm" aria-label="Pages" />
          <UnstyledButton
            component="a"
            href="/"
            aria-label="nabu home"
            onClick={(e) => {
              e.preventDefault();
              onHome();
              close();
            }}
          >
            <Group gap={6} c="var(--mantine-color-text)">
              <Box display="flex" dangerouslySetInnerHTML={{ __html: logo }} />
              <Text fw={700}>nabu</Text>
            </Group>
          </UnstyledButton>
        </Group>
      </AppShell.Header>
      <AppShell.Navbar p="xs">
        <Stack align="center" gap="xs" h="100%">
          {items.map((i) => button(i.label, i.icon, () => onSelect(i.key), i.key === current, i.disabled))}
          <Stack align="center" gap="xs" mt="auto">
            {actions.map((a) => button(a.label, a.icon, a.onClick))}
          </Stack>
        </Stack>
      </AppShell.Navbar>
      <AppShell.Main h="100dvh">{children}</AppShell.Main>
    </AppShell>
  );
}
