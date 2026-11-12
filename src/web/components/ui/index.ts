/**
 * nabu の土台の部品の玄関。@mantine/* と @tabler/icons-react を import してよいのはこの directory だけ（app/ の Provider と Shell は @mantine/* だけ例外）で、
 * components と pages はここから役割（見出し・補足・一覧の行…）を選ぶだけにする。大きさと色は部品が決める。
 * この file は ui の file から出し直すだけで、ライブラリを直に参照しない（素通しにしない）
 */

// 枠
export { ListDetailLayout } from "./ListDetailLayout";
export { Pane } from "./Pane";

// 並べる（間隔と揃え方の props だけを持つ）
export { Page } from "./Page";
export { Stack } from "./Stack";
export { Group } from "./Group";
export { Columns } from "./Columns";
export { Form } from "./Form";
export { Indent } from "./Indent";

// 文字
export { Heading } from "./Heading";
export { Caption } from "./Caption";
export { ErrorText } from "./ErrorText";
export { Timestamp } from "./Timestamp";
export { ExternalLink } from "./ExternalLink";

// 入力と操作
export { Button } from "./Button";
export { IconButton } from "./IconButton";
export { SegmentedControl } from "./SegmentedControl";
export { Checkbox } from "./Checkbox";
export { LineInput } from "./LineInput";
export { LinesInput } from "./LinesInput";
export { DatePickerButton } from "./DatePickerButton";
export { ListRow } from "./ListRow";
export { Pager } from "./Pager";

// 組み立て
export { Section } from "./Section";
export { Passage } from "./Passage";
export { Dialog } from "./Dialog";
export { EmptyState } from "./EmptyState";
export { KeyTable } from "./KeyTable";
export { Prose } from "./Prose";

export * from "./icons";

export * from "./hooks";
