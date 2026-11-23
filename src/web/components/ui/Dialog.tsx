import type { ReactNode } from "react";
import { Modal, type ModalProps, Stack } from "@mantine/core";

/** 前面に出す dialog。size は中身が広い表のときだけ大きくする */
export const Dialog = ({
  title,
  size,
  opened,
  onClose,
  children,
}: {
  title: string;
  size?: ModalProps["size"];
  opened: boolean;
  onClose: () => void;
  children: ReactNode;
}) => (
  <Modal title={title} size={size} opened={opened} onClose={onClose} centered>
    <Stack>{children}</Stack>
  </Modal>
);
