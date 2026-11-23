/**
 * nabu の土台の部品の玄関。@mantine/* と @tabler/icons-react を import してよいのはこの directory だけ（app/ の Provider と Shell は @mantine/* だけ例外）で、
 * components と pages はここから役割（見出し・補足・一覧の行…）を選ぶだけにする。大きさと色は部品が決める。
 * この file は ui の file から出し直すだけで、ライブラリを直に参照しない（素通しにしない）
 */

// 並べる（間隔と揃え方の props だけを持つ）
export { Page } from "./Page";
export { Stack } from "./Stack";
export { SideColumn } from "./SideColumn";
export { Indent } from "./Indent";
export { Group } from "./Group";
export { Form } from "./Form";

// 文字
export { Caption } from "./Caption";
export { ErrorText } from "./ErrorText";
export { Timestamp } from "./Timestamp";
export { ExternalLink } from "./ExternalLink";

// 入力と操作
export { Button } from "./Button";
export { IconButton } from "./IconButton";
export { MoreMenu } from "./MoreMenu";
export { Checkbox } from "./Checkbox";
export { CheckMark } from "./CheckMark";
export { TextButton } from "./TextButton";
export { LineInput } from "./LineInput";
export { LinesInput } from "./LinesInput";
export { Choice } from "./Choice";
export { DatePickerButton } from "./DatePickerButton";
export { CheckFacet, Facet } from "./Facet";
export { FoldRow } from "./FoldRow";
export { FoldTable } from "./FoldTable";
export { Pager } from "./Pager";

// 組み立て
export { PageTitle } from "./PageTitle";
export { Section } from "./Section";
export { Passage } from "./Passage";
export { FoldSections } from "./FoldSections";
export { Dialog } from "./Dialog";
export { EmptyState } from "./EmptyState";
export { KeyTable } from "./KeyTable";
export { StackTable } from "./StackTable";
export { Prose } from "./Prose";

export * from "./icons";

export * from "./hooks";
