import { EmptyState as MantineEmptyState } from "@mantine/core";

/** 何もないことの説明 */
export const EmptyState = ({ title, description }: { title: string; description?: string }) => (
  <MantineEmptyState title={title} description={description} size="sm" py="xl" />
);
