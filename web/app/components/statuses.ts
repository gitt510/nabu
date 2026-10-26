import type { Status } from "../../src/fields";
import { type Choice, CircleCheckIcon, CircleDotIcon, CircleXIcon, InboxIcon } from "./ui";

/** status の名前と icon。Dropped は理由を聞くので、線で区切って下に置く */
export const STATUS_CHOICES: Choice<Status>[] = [
  { value: "inbox", label: "Inbox", icon: InboxIcon },
  { value: "doing", label: "Doing", icon: CircleDotIcon },
  { value: "done", label: "Done", icon: CircleCheckIcon },
  { value: "dropped", label: "Dropped", icon: CircleXIcon, apart: true },
];
