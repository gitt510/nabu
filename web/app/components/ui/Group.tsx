import { Group as MantineGroup, type GroupProps } from "@mantine/core";

/** 横に並べる。受け取るのは間隔・揃え方・折り返しだけで、見た目は変えられない */
export const Group = ({ gap, align, justify, wrap, children }: Pick<GroupProps, "gap" | "align" | "justify" | "wrap" | "children">) => (
  <MantineGroup gap={gap} align={align} justify={justify} wrap={wrap}>
    {children}
  </MantineGroup>
);
