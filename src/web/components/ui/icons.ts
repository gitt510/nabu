// 使う icon だけを、Tabler の形の名前（Icon 接頭辞を外したもの）で出す。何の意味で使うかは使う側が決め、
// 大きさと線の太さは icon を置く部品が決める
import type { ComponentType } from "react";

export type IconComponent = ComponentType<{ size?: number; stroke?: number }>;

export {
  IconArrowLeft as ArrowLeftIcon,
  IconArrowBackUp as ArrowBackUpIcon,
  IconCalendar as CalendarIcon,
  IconChecklist as ChecklistIcon,
  IconChevronLeft as ChevronLeftIcon,
  IconChevronRight as ChevronRightIcon,
  IconCircleCheck as CircleCheckIcon,
  IconCirclePlus as CirclePlusIcon,
  IconCircleX as CircleXIcon,
  IconFolder as FolderIcon,
  IconGitPullRequest as GitPullRequestIcon,
  IconHourglass as HourglassIcon,
  IconKeyboard as KeyboardIcon,
  IconLink as LinkIcon,
  IconMoon as MoonIcon,
  IconNotes as NotesIcon,
  IconPencil as PencilIcon,
  IconSeedling as SeedlingIcon,
  IconSun as SunIcon,
  IconTag as TagIcon,
  IconTicket as TicketIcon,
  IconTrash as TrashIcon,
} from "@tabler/icons-react";
