import type { ComponentType, ReactNode } from "react";
import { ActionIcon, AppShell, Box, Group, Stack, Text, Tooltip, UnstyledButton } from "@mantine/core";
import { useMediaQuery } from "@mantine/hooks";

import logo from "./logo.svg?raw";

type Icon = ComponentType<{ size?: number; stroke?: number }>;

export type RailItem<K extends string> = { key: K; label: string; icon: Icon; disabled?: boolean };
/** page をまたぐ操作。keyboard の操作は、keyboard の無い狭い画面では出さない */
export type RailAction = { label: string; icon: Icon; onClick: () => void; keyboard?: boolean };

/**
 * 画面の枠：上の帯、page の切り替え、本文。広い画面では左の rail が page の切り替え（上）と操作（下）を持つ。
 * 狭い画面では page の切り替えが下の tab bar（icon とラベル）に移り、操作は上の帯の右に移る。
 * logo は origin への link で、click すると onHome を呼ぶ（新しい tab で開く操作はそのまま効く）
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
  const narrow = useMediaQuery("(max-width: 48em)") ?? false;
  const shown = actions.filter((a) => !(narrow && a.keyboard));
  const iconButton = (label: string, Icon: Icon, onClick: () => void, active = false, disabled = false) => (
    <Tooltip key={label} label={disabled ? `${label} (coming soon)` : label} position={narrow ? "bottom" : "right"}>
      <ActionIcon
        size={narrow ? "lg" : "xl"}
        variant={active ? "light" : "subtle"}
        color={active ? undefined : "gray"}
        aria-label={label}
        aria-current={active ? "page" : undefined}
        disabled={disabled}
        onClick={onClick}
      >
        <Icon size={narrow ? 20 : 22} stroke={1.75} />
      </ActionIcon>
    </Tooltip>
  );
  return (
    <AppShell
      header={{ height: 48 }}
      navbar={{ width: 64, breakpoint: "sm", collapsed: { mobile: true } }}
      footer={{ height: 60, collapsed: !narrow }}
    >
      <AppShell.Header>
        <Group h="100%" px="md" justify="space-between">
          <UnstyledButton
            component="a"
            href="/"
            aria-label="nabu home"
            onClick={(e) => {
              e.preventDefault();
              onHome();
            }}
          >
            <Group gap={6} c="var(--mantine-color-text)">
              <Box display="flex" dangerouslySetInnerHTML={{ __html: logo }} />
              <Text fw={700}>nabu</Text>
            </Group>
          </UnstyledButton>
          {narrow ? <Group gap="xs">{shown.map((a) => iconButton(a.label, a.icon, a.onClick))}</Group> : null}
        </Group>
      </AppShell.Header>
      <AppShell.Navbar p="xs">
        <Stack align="center" gap="xs" h="100%">
          {items.map((i) => iconButton(i.label, i.icon, () => onSelect(i.key), i.key === current, i.disabled))}
          <Stack align="center" gap="xs" mt="auto">
            {shown.map((a) => iconButton(a.label, a.icon, a.onClick))}
          </Stack>
        </Stack>
      </AppShell.Navbar>
      <AppShell.Main h="100dvh">{children}</AppShell.Main>
      <AppShell.Footer>
        <Group h="100%" grow gap={0} component="nav" aria-label="Pages">
          {items.map((i) => {
            const active = i.key === current;
            return (
              <UnstyledButton key={i.key} h="100%" disabled={i.disabled} aria-current={active ? "page" : undefined} onClick={() => onSelect(i.key)}>
                <Stack align="center" gap={2} c={i.disabled ? "dimmed" : active ? "var(--mantine-primary-color-filled)" : "var(--mantine-color-text)"} opacity={i.disabled ? 0.5 : 1}>
                  <i.icon size={22} stroke={1.75} />
                  <Text span size="xs" fw={active ? 700 : 400}>
                    {i.label}
                  </Text>
                </Stack>
              </UnstyledButton>
            );
          })}
        </Group>
      </AppShell.Footer>
    </AppShell>
  );
}
