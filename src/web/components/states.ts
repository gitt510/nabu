import type { Status } from "../../api/schema";

/** status の名前 */
export const STATUS_LABEL: Record<Status, string> = {
  todo: "Todo",
  done: "Done",
  dropped: "Dropped",
};
