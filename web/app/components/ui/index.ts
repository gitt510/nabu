/**
 * nabu の土台の部品の玄関。@mantine/* と @tabler/icons-react を import してよいのはこの directory だけで、
 * components と views はここから役割（見出し・補足・一覧の行…）を選ぶだけにする。大きさと色は部品が決める。
 * この file は ui の file から出し直すだけで、ライブラリを直に参照しない（素通しにしない）
 */

// 枠
export { Provider } from "./Provider";
export { Shell, type RailAction, type RailItem } from "./Shell";
export { ListDetail } from "./ListDetail";

// 並べる（間隔と揃え方の props だけを持つ）
export { Page } from "./Page";
export { Stack } from "./Stack";
export { Group } from "./Group";
export { Form } from "./Form";

// 文字
export { Caption, ErrorText, ExternalLink, Heading, Subheading, Tag } from "./text";

// 入力と操作
export { Button, ChoiceMenu, CopyText, IconButton, LinesInput, ListRow, Pager, SearchInput, Segments, Select, Toggle, type Choice } from "./controls";

// 組み立て
export { Dialog, Empty, Facts, KeyTable, Prose } from "./blocks";

export * from "./icons";

export * from "./hooks";
