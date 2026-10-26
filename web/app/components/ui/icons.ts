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
  IconChecklist as ChecklistIcon,
  IconChevronLeft as ChevronLeftIcon,
  IconChevronRight as ChevronRightIcon,
  IconCircleCheck as CircleCheckIcon,
  IconCircleDot as CircleDotIcon,
  IconCircleX as CircleXIcon,
  IconHourglass as HourglassIcon,
  IconInbox as InboxIcon,
  IconKeyboard as KeyboardIcon,
  IconMoon as MoonIcon,
  IconSearch as SearchIcon,
  IconSeedling as SeedlingIcon,
  IconSun as SunIcon,
  IconTrash as TrashIcon,
} from "@tabler/icons-react";
