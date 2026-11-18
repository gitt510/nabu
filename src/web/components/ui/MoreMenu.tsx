import { ActionIcon, Menu, Text } from "@mantine/core";

import { DotsIcon, type IconComponent } from "./icons";

/** 三点の button から開く操作の一覧。各行は icon と名前に、あれば右端に key を添える */
export const MoreMenu = ({
  label,
  items,
}: {
  label: string;
  items: { label: string; icon: IconComponent; hotkey?: string; onClick: () => void }[];
}) => (
  <Menu position="bottom-end" shadow="md" withinPortal>
    <Menu.Target>
      <ActionIcon variant="subtle" color="gray" aria-label={label}>
        <DotsIcon size={18} />
      </ActionIcon>
    </Menu.Target>
    <Menu.Dropdown>
      {items.map(({ label, icon: Icon, hotkey, onClick }) => (
        <Menu.Item
          key={label}
          leftSection={<Icon size={16} />}
          rightSection={
            hotkey ? (
              <Text span size="xs" c="dimmed" ff="monospace">
                {hotkey}
              </Text>
            ) : null
          }
          onClick={onClick}
        >
          {label}
        </Menu.Item>
      ))}
    </Menu.Dropdown>
  </Menu>
);
