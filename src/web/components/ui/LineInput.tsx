import { TextInput } from "@mantine/core";

/** 1 行の入力。横に並んだときは、空いている幅いっぱいに伸びる。onEscape があれば Esc で呼び、その場の編集をやめられる。onBlur は欄の外へ focus が移ったときに呼ぶ */
export const LineInput = ({
  label,
  placeholder,
  value,
  onChange,
  onEscape,
  onBlur,
  autoFocus = false,
}: {
  label: string;
  placeholder?: string;
  value: string;
  onChange: (v: string) => void;
  onEscape?: () => void;
  onBlur?: () => void;
  autoFocus?: boolean;
}) => (
  <TextInput
    flex={1}
    aria-label={label}
    placeholder={placeholder}
    autoComplete="off"
    autoFocus={autoFocus}
    // dialog の中では、Mantine が開いたときにこの印の欄へ focus を移す
    data-autofocus={autoFocus || undefined}
    value={value}
    onChange={(e) => onChange(e.currentTarget.value)}
    onBlur={onBlur}
    onKeyDown={(e) => {
      if (e.key === "Escape" && onEscape) onEscape();
    }}
  />
);
