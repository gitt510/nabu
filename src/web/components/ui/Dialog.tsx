import type { ReactNode } from "react";
import { Modal, Stack } from "@mantine/core";

/** 前面に出す dialog */
export const Dialog = ({ title, opened, onClose, children }: { title: string; opened: boolean; onClose: () => void; children: ReactNode }) => (
  <Modal title={title} opened={opened} onClose={onClose} centered>
    <Stack>{children}</Stack>
  </Modal>
);
