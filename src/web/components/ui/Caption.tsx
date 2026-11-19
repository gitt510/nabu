import type { ReactNode } from "react";
import { Box, Text } from "@mantine/core";

import type { IconComponent } from "./icons";

// icon の下端を文字の baseline からこれだけ下げ、icon の中央を大文字と数字の高さの中央に合わせる。
// 行の箱で中央に揃えると、上下の余白が非対称な Noto Sans JP ではブラウザごとにずれるので、baseline を基準にする
const ICON_DROP = "-0.2em";

/** 補足。日付、件数、空の説明。icon を渡すと、何の補足かを文字の前に示す */
export const Caption = ({
  children,
  mono,
  icon: Icon,
}: {
  children: ReactNode;
  mono?: boolean;
  icon?: IconComponent;
}) => (
  <Text
    span
    size="xs"
    c="dimmed"
    ff={mono ? "monospace" : undefined}
    style={Icon ? { whiteSpace: "nowrap" } : undefined}
  >
    {Icon ? (
      <Box component="span" display="inline-flex" mr={4} style={{ verticalAlign: ICON_DROP }}>
        <Icon size={14} />
      </Box>
    ) : null}
    {children}
  </Text>
);
