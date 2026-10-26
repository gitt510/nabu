import type { ReactNode } from "react";
import { Box, EmptyState, Group, Kbd, Modal, Stack, Table, Typography } from "@mantine/core";

import { Caption } from "./text";

/** markdown から作った HTML を、本文の書式で出す */
export const Prose = ({ html }: { html: string }) => (
  <Typography>
    <Box dangerouslySetInnerHTML={{ __html: html }} />
  </Typography>
);

/** 名前と値の組を横に流す（task の properties） */
export const Facts = ({ items }: { items: { label: string; value: ReactNode }[] }) => (
  <Group gap="sm" component="dl" my={0}>
    {items.map((i) => (
      <Group key={i.label} gap={6} wrap="nowrap">
        <Box component="dt">
          <Caption mono>{i.label}</Caption>
        </Box>
        <Group component="dd" gap={4} ms={0}>
          {i.value}
        </Group>
      </Group>
    ))}
  </Group>
);

/** 何もないことの説明 */
export const Empty = ({ title, description }: { title: string; description?: string }) => <EmptyState title={title} description={description} size="sm" py="xl" />;

/** 前面に出す dialog */
export const Dialog = ({ title, opened, onClose, children }: { title: string; opened: boolean; onClose: () => void; children: ReactNode }) => (
  <Modal title={title} opened={opened} onClose={onClose} centered>
    <Stack>{children}</Stack>
  </Modal>
);

/** key と、その働きの一覧 */
export const KeyTable = ({ rows }: { rows: { keys: string[]; label: string }[] }) => (
  <Table>
    <Table.Tbody>
      {rows.map((r) => (
        <Table.Tr key={r.label}>
          <Table.Td>
            <Group gap={4} wrap="nowrap">
              {r.keys.map((k) => (
                <Kbd key={k} size="xs">
                  {k}
                </Kbd>
              ))}
            </Group>
          </Table.Td>
          <Table.Td>{r.label}</Table.Td>
        </Table.Tr>
      ))}
    </Table.Tbody>
  </Table>
);
