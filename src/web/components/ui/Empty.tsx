import { EmptyState } from "@mantine/core";

/** 何もないことの説明 */
export const Empty = ({ title, description }: { title: string; description?: string }) => <EmptyState title={title} description={description} size="sm" py="xl" />;
