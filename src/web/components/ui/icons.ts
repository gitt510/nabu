// 使う icon だけを、Tabler の形の名前（Icon 接頭辞を外したもの）で出す。何の意味で使うかは使う側が決め、
// 大きさと線の太さは icon を置く部品が決める
import type { ComponentType } from "react";

export type IconComponent = ComponentType<{ size?: number; stroke?: number }>;

export {
  IconArrowBackUp as ArrowBackUpIcon,
  IconCalendar as CalendarIcon,
  IconCheck as CheckIcon,
  IconChecklist as ChecklistIcon,
  IconChevronDown as ChevronDownIcon,
  IconChevronLeft as ChevronLeftIcon,
  IconChevronRight as ChevronRightIcon,
  IconChevronUp as ChevronUpIcon,
  IconCircleCheck as CircleCheckIcon,
  IconCircleX as CircleXIcon,
  IconDots as DotsIcon,
  IconFolder as FolderIcon,
  IconGitPullRequest as GitPullRequestIcon,
  IconHome as HomeIcon,
  IconHourglass as HourglassIcon,
  IconKeyboard as KeyboardIcon,
  IconLink as LinkIcon,
  IconMoon as MoonIcon,
  IconNotes as NotesIcon,
  IconPencil as PencilIcon,
  IconPlus as PlusIcon,
  IconSeedling as SeedlingIcon,
  IconStack2 as Stack2Icon,
  IconSun as SunIcon,
  IconTag as TagIcon,
  IconTicket as TicketIcon,
  IconTrash as TrashIcon,
} from "@tabler/icons-react";
