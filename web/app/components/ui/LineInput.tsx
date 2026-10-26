import { TextInput } from "@mantine/core";

/** 1 行の入力。横に並んだときは、空いている幅いっぱいに伸びる。onEscape があれば Esc で呼び、その場の編集をやめられる */
export const LineInput = ({
  label,
  placeholder,
  value,
  onChange,
  onEscape,
  autoFocus = false,
}: {
  label: string;
  placeholder?: string;
  value: string;
  onChange: (v: string) => void;
  onEscape?: () => void;
  autoFocus?: boolean;
}) => (
  <TextInput
    flex={1}
    aria-label={label}
    placeholder={placeholder}
    autoComplete="off"
    autoFocus={autoFocus}
    value={value}
    onChange={(e) => onChange(e.currentTarget.value)}
    onKeyDown={(e) => {
      if (e.key === "Escape" && onEscape) onEscape();
    }}
  />
);
