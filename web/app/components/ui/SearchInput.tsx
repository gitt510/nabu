import type { Ref } from "react";
import { TextInput } from "@mantine/core";

import { SearchIcon } from "./icons";

/** 一覧の上の検索欄 */
export const SearchInput = ({ value, onChange, ref }: { value: string; onChange: (v: string) => void; ref?: Ref<HTMLInputElement> }) => (
  <TextInput
    ref={ref}
    type="search"
    size="sm"
    placeholder="Search"
    aria-label="Search"
    autoComplete="off"
    leftSection={<SearchIcon size={16} />}
    value={value}
    onChange={(e) => onChange(e.currentTarget.value)}
  />
);
