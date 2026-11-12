import type { ReactNode } from "react";
import { Box } from "@mantine/core";

/** 中身を 1 段下げる。木の子を親の下に並べるときに使う */
export const Indent = ({ children }: { children: ReactNode }) => <Box pl="lg">{children}</Box>;
