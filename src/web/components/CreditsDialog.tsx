import { Dialog, StackTable } from "./ui";

/** この page を作った主な道具。何で作ったかを見返すためのもの */
const STACK = [
  { layer: "UI", label: "React", href: "https://react.dev", role: "Components and state" },
  { layer: "UI", label: "Mantine", href: "https://mantine.dev", role: "Layout, inputs, dialogs" },
  {
    layer: "UI",
    label: "TanStack Table",
    href: "https://tanstack.com/table",
    role: "The task table and its facets",
  },
  { layer: "UI", label: "Tabler Icons", href: "https://tabler.io/icons", role: "Icons" },
  { layer: "API", label: "Hono", href: "https://hono.dev", role: "Routes and OpenAPI schema" },
  {
    layer: "Hosting",
    label: "Cloudflare Workers",
    href: "https://developers.cloudflare.com/workers/",
    role: "Serves the page and the API",
  },
  {
    layer: "Data",
    label: "Cloudflare D1",
    href: "https://developers.cloudflare.com/d1/",
    role: "Stores tasks and habits",
  },
  { layer: "Build", label: "Vite", href: "https://vite.dev", role: "Dev server and bundling" },
];

export const CreditsDialog = ({ opened, onClose }: { opened: boolean; onClose: () => void }) => (
  <Dialog title="Built with" size="lg" opened={opened} onClose={onClose}>
    <StackTable rows={STACK} />
  </Dialog>
);
