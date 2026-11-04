import type { ReactNode } from "react";
import { NavLink } from "@mantine/core";

/** 一覧の 1 行。左端の印（チェックボックスなど）、題、右端の補足 */
export const ListRow = ({ id, lead, label, aside, active, onClick }: { id: string; lead?: ReactNode; label: ReactNode; aside: ReactNode; active: boolean; onClick: () => void }) => (
  <NavLink id={id} leftSection={lead} label={label} rightSection={aside} active={active} onClick={onClick} noWrap variant="light" />
);
