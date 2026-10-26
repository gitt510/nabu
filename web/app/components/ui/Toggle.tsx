import { Chip } from "@mantine/core";

import { type IconComponent } from "./icons";

/** on / off の絞り込み */
export const Toggle = ({ label, icon: Icon, checked, onChange }: { label: string; icon: IconComponent; checked: boolean; onChange: (v: boolean) => void }) => (
  <Chip size="xs" variant="outline" checked={checked} onChange={onChange} icon={<Icon size={12} />}>
    {label}
  </Chip>
);
