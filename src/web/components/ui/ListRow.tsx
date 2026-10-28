import type { ReactNode } from "react";
import { NavLink } from "@mantine/core";

/** 一覧の 1 行。題と、右端の補足 */
export const ListRow = ({ id, label, aside, active, onClick }: { id: string; label: ReactNode; aside: ReactNode; active: boolean; onClick: () => void }) => (
  <NavLink id={id} label={label} rightSection={aside} active={active} onClick={onClick} noWrap variant="light" />
);
