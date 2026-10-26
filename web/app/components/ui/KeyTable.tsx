import { Group, Kbd, Table } from "@mantine/core";

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
