import { CopyButton, Group, Text, Tooltip, UnstyledButton } from "@mantine/core";

import { CheckIcon, CopyIcon } from "./icons";

/** click で値を copy する文字列（task の番号など）。文字の大きさは置いた場所に合わせ、icon で押せると示す */
export const CopyText = ({ value }: { value: string }) => (
  <CopyButton value={value}>
    {({ copied, copy }) => (
      <Tooltip label={copied ? "Copied" : "Click to copy"}>
        <UnstyledButton onClick={copy} display="inline-block">
          <Group component="span" gap={4} wrap="nowrap">
            <Text span inherit c={copied ? "teal" : "dimmed"} fw={400}>
              {value}
            </Text>
            <Text span c="dimmed" display="inline-flex">
              {copied ? <CheckIcon size={16} /> : <CopyIcon size={16} />}
            </Text>
          </Group>
        </UnstyledButton>
      </Tooltip>
    )}
  </CopyButton>
);
