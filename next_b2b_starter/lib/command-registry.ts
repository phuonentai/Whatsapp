import type { LucideIcon } from "lucide-react";
import {
  BookOpen,
  Bot,
  Boxes,
  Contact,
  CreditCard,
  Inbox,
  Instagram,
  LayoutDashboard,
  MessageCircle,
  ScrollText,
  Settings,
  ShieldCheck,
  User,
  Users,
} from "lucide-react";

export interface CommandDestination {
  id: string;
  title: string;
  url: string;
  section: string;
  icon: LucideIcon;
  keywords?: string[];
}

/**
 * Command-palette destinations: every sidebar entry plus every settings
 * detail view. Consumed by the command palette for ⌘K navigation and by the
 * global shortcuts hook for `g <key>` navigation.
 */
export const commandRegistry: CommandDestination[] = [
  {
    id: "dashboard",
    title: "Dashboard",
    url: "/dashboard",
    section: "Navigate",
    icon: LayoutDashboard,
    keywords: ["home", "inicio"],
  },
  {
    id: "inbox",
    title: "Inbox",
    url: "/dashboard/inbox",
    section: "Navigate",
    icon: Inbox,
    keywords: ["conversations", "conversaciones", "buzón", "mensajes", "chats"],
  },
  {
    id: "crm",
    title: "CRM",
    url: "/dashboard/crm",
    section: "Navigate",
    icon: Contact,
    keywords: ["contactos", "contacts", "empresas", "negocios", "deals"],
  },
  {
    id: "knowledge",
    title: "Knowledge Base",
    url: "/dashboard/knowledge",
    section: "Navigate",
    icon: BookOpen,
    keywords: ["conocimiento", "documentos", "documents", "base"],
  },
  {
    id: "settings",
    title: "Settings",
    url: "/dashboard/settings",
    section: "Navigate",
    icon: Settings,
    keywords: ["ajustes", "configuración", "configuration"],
  },
  {
    id: "settings-account",
    title: "Account",
    url: "/dashboard/settings?view=profile",
    section: "Settings",
    icon: User,
    keywords: ["perfil", "profile", "workspace"],
  },
  {
    id: "settings-team",
    title: "Team",
    url: "/dashboard/settings?view=members",
    section: "Settings",
    icon: Users,
    keywords: ["members", "miembros", "invite", "invitar", "roles"],
  },
  {
    id: "settings-subscription",
    title: "Subscription",
    url: "/dashboard/settings?view=subscription",
    section: "Settings",
    icon: CreditCard,
    keywords: ["billing", "plan", "facturación", "polar"],
  },
  {
    id: "settings-modules",
    title: "Modules",
    url: "/dashboard/settings?view=modules",
    section: "Settings",
    icon: Boxes,
    keywords: ["módulos", "features"],
  },
  {
    id: "settings-ai",
    title: "AI Copilot",
    url: "/dashboard/settings?view=ai",
    section: "Settings",
    icon: Bot,
    keywords: ["asistente", "agent", "ia"],
  },
  {
    id: "settings-compliance",
    title: "Compliance",
    url: "/dashboard/settings?view=compliance",
    section: "Settings",
    icon: ShieldCheck,
    keywords: ["ley 1581", "data", "datos", "habeas data"],
  },
  {
    id: "settings-messaging",
    title: "Messaging",
    url: "/dashboard/settings?view=whatsapp",
    section: "Settings",
    icon: MessageCircle,
    keywords: ["whatsapp", "canal", "channel", "mensajería"],
  },
  {
    id: "settings-instagram",
    title: "Instagram",
    url: "/dashboard/settings?view=instagram",
    section: "Settings",
    icon: Instagram,
    keywords: ["ig", "dms", "red social"],
  },
  {
    id: "settings-audit",
    title: "Audit log",
    url: "/dashboard/settings?view=audit",
    section: "Settings",
    icon: ScrollText,
    keywords: ["auditoría", "actividad", "activity", "log"],
  },
];

export interface GlobalShortcut {
  id: string;
  keys: string[];
  label: string;
  url?: string;
}

/**
 * Global keyboard shortcuts rendered by the help overlay and handled by
 * `useGlobalShortcuts`. `g <key>` sequences navigate to the matching
 * destination; `?` opens the help overlay.
 */
export const globalShortcuts: GlobalShortcut[] = [
  { id: "dashboard", keys: ["g", "d"], label: "Go to Dashboard", url: "/dashboard" },
  { id: "inbox", keys: ["g", "i"], label: "Go to Inbox", url: "/dashboard/inbox" },
  { id: "crm", keys: ["g", "c"], label: "Go to CRM", url: "/dashboard/crm" },
  { id: "knowledge", keys: ["g", "k"], label: "Go to Knowledge", url: "/dashboard/knowledge" },
  { id: "settings", keys: ["g", "s"], label: "Go to Settings", url: "/dashboard/settings" },
  { id: "help", keys: ["?"], label: "Shortcuts help" },
  { id: "palette", keys: ["⌘", "K"], label: "Command palette" },
];
