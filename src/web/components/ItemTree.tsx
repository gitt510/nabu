import type { ReactNode } from "react";

import type { Item } from "../../api/schema";
import { childrenOf } from "../outline";
import { Indent, Stack } from "./ui";

/** roots とその下の行を、子を 1 段下げて並べる。1 行の見た目は row が決める */
export function ItemTree({
  items,
  roots,
  row,
}: {
  items: Item[];
  roots: Item[];
  row: (item: Item) => ReactNode;
}) {
  const branch = (item: Item): ReactNode => {
    const kids = childrenOf(items, item.id);
    return (
      <Stack key={item.id} gap={4}>
        {row(item)}
        {kids.length ? (
          <Indent>
            <Stack gap={4}>{kids.map(branch)}</Stack>
          </Indent>
        ) : null}
      </Stack>
    );
  };
  return <Stack gap="xs">{roots.map(branch)}</Stack>;
}
