import type { ReactNode } from "react";
import { ActionIcon, Button as MantineButton, Group, Title, Tooltip } from "@mantine/core";

import { ChevronLeftIcon, ChevronRightIcon } from "./icons";

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
      {[{ ...prev, icon: ChevronLeftIcon }, { ...next, icon: ChevronRightIcon }].map((b) => (
        <Tooltip key={b.label} label={b.label}>
          <ActionIcon variant="default" size="lg" aria-label={b.label} onClick={b.onClick}>
            <b.icon size={18} />
          </ActionIcon>
        </Tooltip>
      ))}
    </Group>
  </Group>
);
