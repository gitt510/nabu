import type { ReactNode } from "react";
import { Accordion, Group, Text } from "@mantine/core";

/**
 * 区切り線で分けた、題を押して開く段の並び。最初はどれも畳んでおき、いくつでも同時に開ける。
 * hint は題の右に薄く添える、畳んだままでも分かる中身の要約（入れた行の数など）
 */
export const FoldSections = ({
  items,
}: {
  items: { value: string; title: string; hint?: string; children: ReactNode }[];
}) => (
  <Accordion multiple chevronPosition="left">
    {items.map((i) => (
      <Accordion.Item key={i.value} value={i.value}>
        <Accordion.Control>
          <Group gap="xs" wrap="nowrap">
            <Text span size="sm" fw={700}>
              {i.title}
            </Text>
            {i.hint ? (
              <Text span size="xs" c="dimmed">
                {i.hint}
              </Text>
            ) : null}
          </Group>
        </Accordion.Control>
        <Accordion.Panel>{i.children}</Accordion.Panel>
      </Accordion.Item>
    ))}
  </Accordion>
);
