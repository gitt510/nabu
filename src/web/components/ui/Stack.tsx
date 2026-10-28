import { Stack as MantineStack, type StackProps } from "@mantine/core";

/** 縦に並べる。受け取るのは間隔と揃え方だけで、見た目は変えられない */
export const Stack = ({ gap, align, justify, children }: Pick<StackProps, "gap" | "align" | "justify" | "children">) => (
  <MantineStack gap={gap} align={align} justify={justify}>
    {children}
  </MantineStack>
);
