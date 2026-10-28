import { ActionIcon, Tooltip } from "@mantine/core";

import { type IconComponent } from "./icons";

/** icon だけの button。説明は tooltip に出す */
export const IconButton = ({ label, icon: Icon, onClick }: { label: string; icon: IconComponent; onClick: () => void }) => (
  <Tooltip label={label}>
    <ActionIcon variant="subtle" color="gray" aria-label={label} onClick={onClick}>
      <Icon size={18} />
    </ActionIcon>
  </Tooltip>
);
