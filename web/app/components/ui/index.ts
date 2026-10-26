/**
 * nabu の土台の部品。@mantine/* と @tabler/icons-react を import してよいのはこの directory だけで、
 * components と views はここから役割（見出し・補足・一覧の行…）を選ぶだけにする。大きさと色は部品が決める
 */

// 枠
export { Provider } from "./Provider";
export { Shell, type RailAction, type RailItem } from "./Shell";
export { ListDetail } from "./ListDetail";

// 並べる（layout の props だけを持つ）
export { Box, Group, Stack } from "@mantine/core";

// 文字
export { Caption, ErrorText, ExternalLink, Heading, Subheading, Tag } from "./text";

// 入力と操作
export { Button, ChoiceMenu, CopyText, IconButton, LinesInput, ListRow, Pager, SearchInput, Segments, Select, Toggle, type Choice } from "./controls";

// 組み立て
export { Dialog, Empty, Facts, KeyTable, Prose } from "./blocks";

export * from "./icons";

// hooks
export { useComputedColorScheme, useMantineColorScheme } from "@mantine/core";
export { useHotkeys, useLocalStorage, useMediaQuery } from "@mantine/hooks";
