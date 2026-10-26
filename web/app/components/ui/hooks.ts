// 使う hook だけを、ライブラリから出し直す。index.ts はライブラリを直に参照しない
export { useComputedColorScheme, useMantineColorScheme } from "@mantine/core";
export { useHotkeys, useLocalStorage, useMediaQuery } from "@mantine/hooks";
