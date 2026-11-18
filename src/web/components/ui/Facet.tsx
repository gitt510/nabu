import type { ReactNode } from "react";
import { Box, Checkbox, NavLink, Stack, Text } from "@mantine/core";

import { CheckIcon } from "./icons";

type Option = { value: string; label: string; count: number };

const Title = ({ children }: { children: string }) => (
  <Text size="xs" fw={700} c="dimmed" px="sm" pb={4}>
    {children}
  </Text>
);

const Row = ({
  option,
  mark,
  checked,
  role,
  onClick,
}: {
  option: Option;
  mark: ReactNode;
  checked: boolean;
  role: "radio" | "checkbox";
  onClick: () => void;
}) => (
  <NavLink
    label={option.label}
    leftSection={mark}
    rightSection={
      <Text span size="xs" c="dimmed" ff="monospace">
        {option.count}
      </Text>
    }
    role={role}
    aria-checked={checked}
    onClick={onClick}
    py={4}
  />
);

/**
 * 一覧を絞る軸のうち、1 つだけ選ぶもの。題の下に値を縦に並べ、選んだ値の左に ✓ を付け、右にそれぞれに当てはまる数を添える。
 * 背景は変えない
 */
export const Facet = ({
  title,
  data,
  value,
  onChange,
}: {
  title: string;
  data: Option[];
  value: string;
  onChange: (v: string) => void;
}) => (
  <Stack gap={2} role="radiogroup" aria-label={title}>
    <Title>{title}</Title>
    {data.map((d) => (
      <Row
        key={d.value}
        option={d}
        role="radio"
        checked={d.value === value}
        // ✓ の無い行も同じ幅を空け、文字の頭を揃える
        mark={<Box w={16}>{d.value === value ? <CheckIcon size={16} /> : null}</Box>}
        onClick={() => onChange(d.value)}
      />
    ))}
  </Stack>
);

/**
 * 一覧を絞る軸のうち、いくつも選べるもの。値ごとに checkbox を付け、右にそれぞれに当てはまる数を添える。
 * 何も選んでいなければ絞らない（All の行は置かない）
 */
export const CheckFacet = ({
  title,
  data,
  value,
  onChange,
}: {
  title: string;
  data: Option[];
  value: string[];
  onChange: (v: string[]) => void;
}) => (
  <Stack gap={2} role="group" aria-label={title}>
    <Title>{title}</Title>
    {data.map((d) => {
      const on = value.includes(d.value);
      return (
        <Row
          key={d.value}
          option={d}
          role="checkbox"
          checked={on}
          // 行ごと押して付け外しするので、checkbox は印として見せるだけ
          mark={
            <Checkbox
              size="xs"
              checked={on}
              readOnly
              tabIndex={-1}
              aria-hidden
              style={{ pointerEvents: "none" }}
            />
          }
          onClick={() => onChange(on ? value.filter((v) => v !== d.value) : [...value, d.value])}
        />
      );
    })}
  </Stack>
);
