import { Box, Typography } from "@mantine/core";

/** markdown から作った HTML を、本文の書式で出す */
export const Prose = ({ html }: { html: string }) => (
  <Typography>
    <Box dangerouslySetInnerHTML={{ __html: html }} />
  </Typography>
);
