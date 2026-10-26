import { CopyButton, Text, Tooltip, UnstyledButton } from "@mantine/core";

/** click で値を copy する文字列（slug など） */
export const CopyText = ({ value }: { value: string }) => (
  <CopyButton value={value}>
    {({ copied, copy }) => (
      <Tooltip label={copied ? "Copied" : "Click to copy"}>
        <UnstyledButton onClick={copy}>
          <Text span size="sm" c="dimmed" ff="monospace">
            {value}
          </Text>
        </UnstyledButton>
      </Tooltip>
    )}
  </CopyButton>
);
