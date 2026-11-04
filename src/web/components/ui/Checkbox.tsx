import { Checkbox as MantineCheckbox } from "@mantine/core";

import { XIcon } from "./icons";

const Cross = ({ className }: { className?: string }) => <XIcon className={className} stroke={3} />;

/**
 * やった / まだ、を付け外しする。bare は題を出さず、一覧の行の頭に置く（行を開く click には伝えない）。
 * crossed はやめた印の ✕ で、ここでは外せない
 */
export const Checkbox = ({ label, checked, onChange, bare, crossed }: { label: string; checked: boolean; onChange: (v: boolean) => void; bare?: boolean; crossed?: boolean }) => (
  <MantineCheckbox
    label={bare ? undefined : label}
    aria-label={bare ? label : undefined}
    checked={checked || crossed}
    icon={crossed ? Cross : undefined}
    color={crossed ? "gray" : undefined}
    readOnly={crossed}
    onChange={(e) => !crossed && onChange(e.currentTarget.checked)}
    onClick={bare ? (e) => e.stopPropagation() : undefined}
  />
);
