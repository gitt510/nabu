import type { ReactNode } from "react";
import { Box } from "@mantine/core";

/** 送信できる入力のまとまり。横に並んだときは、空いている幅いっぱいに伸びる。Enter での送信は browser に任せ、page の再読み込みだけを止める */
export const Form = ({ onSubmit, children }: { onSubmit: () => void; children: ReactNode }) => (
  <Box
    component="form"
    flex={1}
    onSubmit={(e) => {
      e.preventDefault();
      onSubmit();
    }}
  >
    {children}
  </Box>
);
