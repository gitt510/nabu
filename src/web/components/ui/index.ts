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
export { Columns } from "./Columns";
export { Form } from "./Form";

// 文字
export { Heading } from "./Heading";
export { Caption } from "./Caption";
export { ErrorText } from "./ErrorText";
export { Tag } from "./Tag";
export { Stamp } from "./Stamp";
export { ExternalLink } from "./ExternalLink";

// 入力と操作
export { Button } from "./Button";
export { IconButton } from "./IconButton";
export { SearchInput } from "./SearchInput";
export { Segments } from "./Segments";
export { Select } from "./Select";
export { Toggle } from "./Toggle";
export { Checkbox } from "./Checkbox";
export { LineInput } from "./LineInput";
export { LinesInput } from "./LinesInput";
export { CopyText } from "./CopyText";
export { ChoiceMenu, type Choice } from "./ChoiceMenu";
export { DateMenu } from "./DateMenu";
export { ListRow } from "./ListRow";
export { Pager } from "./Pager";

// 組み立て
export { Section } from "./Section";
export { Dialog } from "./Dialog";
export { Empty } from "./Empty";
export { KeyTable } from "./KeyTable";
export { Prose } from "./Prose";

export * from "./icons";

export * from "./hooks";
