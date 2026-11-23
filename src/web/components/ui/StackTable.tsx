import { Table } from "@mantine/core";

import { ExternalLink } from "./ExternalLink";

/** 作るのに使った道具の一覧。層、道具（公式への link）、何に使っているか */
export const StackTable = ({
  rows,
}: {
  rows: { layer: string; label: string; href: string; role: string }[];
}) => (
  <Table>
    <Table.Thead>
      <Table.Tr>
        <Table.Th>Layer</Table.Th>
        <Table.Th>Tool</Table.Th>
        <Table.Th>Used for</Table.Th>
      </Table.Tr>
    </Table.Thead>
    <Table.Tbody>
      {rows.map((r) => (
        <Table.Tr key={r.label}>
          <Table.Td>{r.layer}</Table.Td>
          <Table.Td>
            <ExternalLink href={r.href}>{r.label}</ExternalLink>
          </Table.Td>
          <Table.Td>{r.role}</Table.Td>
        </Table.Tr>
      ))}
    </Table.Tbody>
  </Table>
);
