import type { Item } from "../api/schema";

/** parent の下の行を position 順に。parent が null なら root */
export const childrenOf = (items: Item[], parent: number | null) =>
  items.filter((i) => i.parent_id === parent).sort((a, b) => a.position - b.position);

/** 子のない行は自分の done、子のある行は子が全部 done のとき done */
export const isDone = (items: Item[], item: Item): boolean => {
  const kids = childrenOf(items, item.id);
  return kids.length ? kids.every((k) => isDone(items, k)) : item.done;
};

/** item とその下で、子のない行。親を付け外しするときは、これを全部付け外す */
export const leavesUnder = (items: Item[], item: Item): Item[] => {
  const kids = childrenOf(items, item.id);
  return kids.length ? kids.flatMap((k) => leavesUnder(items, k)) : [item];
};
