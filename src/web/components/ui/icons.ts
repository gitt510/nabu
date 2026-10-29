// 使う icon だけを、Tabler の形の名前（Icon 接頭辞を外したもの）で出す。何の意味で使うかは使う側が決め、
// 大きさと線の太さは icon を置く部品が決める
import type { ComponentType } from "react";

export type IconComponent = ComponentType<{ size?: number; stroke?: number }>;

export {
  IconArrowDown as ArrowDownIcon,
  IconArrowLeft as ArrowLeftIcon,
  IconArrowUp as ArrowUpIcon,
  IconCalendar as CalendarIcon,
  IconCheck as CheckIcon,
  IconChevronDown as ChevronDownIcon,
  IconChecklist as ChecklistIcon,
  IconChevronLeft as ChevronLeftIcon,
  IconChevronRight as ChevronRightIcon,
  IconCircleCheck as CircleCheckIcon,
  IconCircleDot as CircleDotIcon,
  IconCirclePlus as CirclePlusIcon,
  IconCircleX as CircleXIcon,
  IconCopy as CopyIcon,
  IconGitPullRequest as GitPullRequestIcon,
  IconHourglass as HourglassIcon,
  IconInbox as InboxIcon,
  IconKeyboard as KeyboardIcon,
  IconLink as LinkIcon,
  IconMoon as MoonIcon,
  IconPencil as PencilIcon,
  IconSearch as SearchIcon,
  IconSeedling as SeedlingIcon,
  IconSun as SunIcon,
  IconTicket as TicketIcon,
  IconTrash as TrashIcon,
} from "@tabler/icons-react";
