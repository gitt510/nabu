import { Textarea } from "@mantine/core";
import { getHotkeyHandler } from "@mantine/hooks";

/** 複数行の入力。hideLabel なら題は見せず読み上げにだけ渡す（題が外にあるとき）。Enter は改行なので、⌘Enter（Ctrl+Enter）で囲んでいる form を送る。変換中の Enter は送らない */
export const LinesInput = ({
  label,
  hideLabel = false,
  description,
  value,
  onChange,
  error,
}: {
  label: string;
  hideLabel?: boolean;
  description: string;
  value: string;
  onChange: (v: string) => void;
  error?: string;
}) => (
  <Textarea
    label={hideLabel ? undefined : label}
    aria-label={hideLabel ? label : undefined}
    description={description}
    value={value}
    onChange={(e) => onChange(e.currentTarget.value)}
    onKeyDown={(e) => {
      if (e.nativeEvent.isComposing) return;
      getHotkeyHandler([["mod+Enter", () => e.currentTarget.form?.requestSubmit()]])(e);
    }}
    error={error}
    autosize
    minRows={3}
    data-autofocus
  />
);
