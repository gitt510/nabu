import type { ReactNode } from "react";
import {
  ActionIcon,
  AppShell,
  Box,
  Group,
  Stack,
  Text,
  Tooltip,
  UnstyledButton,
} from "@mantine/core";
import { useMediaQuery } from "@mantine/hooks";

import type { IconComponent } from "../components/ui";
import logo from "./logo.svg?raw";

export type RailItem<K extends string> = {
  key: K;
  label: string;
  icon: IconComponent;
};
/** page をまたぐ操作。keyboard の操作は、keyboard の無い狭い画面では出さない */
export type RailAction = {
  label: string;
  icon: IconComponent;
  onClick: () => void;
  keyboard?: boolean;
};

/**
 * 画面の枠：上の帯、page の切り替え、本文。上の帯は page をまたぐもの（logo と操作）を持ち、
 * page の切り替えは広い画面では左の rail、狭い画面では下の tab bar（icon とラベル）に置く。
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
  const iconButton = (
    label: string,
    Icon: IconComponent,
    onClick: () => void,
    active = false,
    header = false,
  ) => (
    <Tooltip key={label} label={label} position={header ? "bottom" : "right"}>
      <ActionIcon
        size={header || narrow ? "lg" : "xl"}
        variant={active ? "light" : "subtle"}
        color={active ? undefined : "gray"}
        aria-label={label}
        aria-current={active ? "page" : undefined}
        onClick={onClick}
      >
        <Icon size={header || narrow ? 20 : 22} stroke={1.75} />
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
        <Group h="100%" px="md" justify="space-between" wrap="nowrap">
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
          <Group gap="xs" wrap="nowrap">
            {shown.map((a) => iconButton(a.label, a.icon, a.onClick, false, true))}
          </Group>
        </Group>
      </AppShell.Header>
      {/* 隠れている側は描かない。AppShell は畳んだ枠を画面の外へ動かすだけなので、残すと Tab や読み上げが届いてしまう */}
      {narrow ? null : (
        <AppShell.Navbar p="xs">
          <Stack align="center" gap="xs" component="nav" aria-label="Pages">
            {items.map((i) =>
              iconButton(i.label, i.icon, () => onSelect(i.key), i.key === current),
            )}
          </Stack>
        </AppShell.Navbar>
      )}
      <AppShell.Main h="100dvh">{children}</AppShell.Main>
      {narrow ? (
        <AppShell.Footer>
          <Group h="100%" grow gap={0} component="nav" aria-label="Pages">
            {items.map((i) => {
              const active = i.key === current;
              return (
                <UnstyledButton
                  key={i.key}
                  h="100%"
                  aria-current={active ? "page" : undefined}
                  onClick={() => onSelect(i.key)}
                >
                  <Stack
                    align="center"
                    gap={2}
                    c={active ? "var(--mantine-primary-color-filled)" : "var(--mantine-color-text)"}
                  >
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
      ) : null}
    </AppShell>
  );
}
