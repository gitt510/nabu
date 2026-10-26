import { Tabs } from "@mantine/core";

/** 1 つだけ選ぶ横並びの切り替え（status など）。中身の panel は持たず、選んだ値だけを返す */
export const Segments = <V extends string>({ items, value, onChange }: { items: { value: V; label: string }[]; value: V; onChange: (v: V) => void }) => (
  <Tabs value={value} onChange={(v) => v && onChange(v as V)}>
    <Tabs.List grow>
      {items.map((i) => (
        <Tabs.Tab key={i.value} value={i.value} px="xs">
          {i.label}
        </Tabs.Tab>
      ))}
    </Tabs.List>
  </Tabs>
);
