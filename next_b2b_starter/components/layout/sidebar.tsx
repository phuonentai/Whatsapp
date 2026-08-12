// components/layout/sidebar.tsx
"use client";

import { useMemo, useState, useEffect, useRef } from "react";
import Link from "next/link";
import { usePathname, useSearchParams } from "next/navigation";

import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import {
  LayoutDashboard,
  Settings,
  Menu,
  X,
  BookOpen,
  MessagesSquare,
  Contact,
  BarChart3,
  Truck,
  Receipt,
  CreditCard,
  Sparkles,
  GraduationCap,
  CalendarClock,
} from "lucide-react";
import { useSidebarStore } from "@/lib/stores/sidebar-store";
import { PRODUCT_NAME } from "@/lib/brand";
import type { ServerPermissions } from "@/lib/auth/server-permissions";
import { useEntitlementQuery } from "@/lib/hooks/use-entitlement";
import type { LucideIcon } from "lucide-react";
import { LogoMark } from "@/components/marketing/site-header";
import { ui } from "@/lib/copy/ui";
// Canonical surface→gate mapping shared with the "Equipo y permisos" impact
// preview (lib/auth/surface-gating.ts): changing a gate here keeps the
// navigation and the preview in sync.
import { SURFACE_GATES } from "@/lib/auth/surface-gating";

interface NavigationItem {
  name: string;
  href: string;
  icon: LucideIcon;
  permission?: string;
  anyPermissions?: string[];
  entitlementKeys?: string[];
  /** Query key used to mark the item active (settings views). */
  queryKey?: string;
  queryValue?: string;
}

const mainNavigation: NavigationItem[] = [
  {
    name: ui.layout.navDashboard,
    href: "/dashboard",
    icon: LayoutDashboard,
    // No permission required - everyone can see the dashboard.
  },
  {
    name: ui.layout.navConversations,
    href: "/dashboard/inbox",
    icon: MessagesSquare,
    // Page enforces org:manage; keep the entry aligned with that gate.
    permission: SURFACE_GATES.inbox.permission,
  },
  {
    name: ui.layout.navContacts,
    href: "/dashboard/crm",
    icon: Contact,
    // Show only when at least one CRM feature is entitled.
    entitlementKeys: SURFACE_GATES.contacts.anyEntitlements,
  },
  {
    name: ui.layout.navInvoices,
    href: "/dashboard/settings?view=siigo",
    icon: Receipt,
    queryKey: "view",
    queryValue: "siigo",
    // Invoicing lives in settings; the section enforces org:manage.
    permission: SURFACE_GATES.invoices.permission,
  },
  {
    name: ui.layout.navPayments,
    href: "/dashboard/settings?view=subscription",
    icon: CreditCard,
    queryKey: "view",
    queryValue: "subscription",
    // Billing section enforces org:manage.
    permission: SURFACE_GATES.payments.permission,
  },
  {
    name: ui.layout.navAnalytics,
    href: "/dashboard/reportes",
    icon: BarChart3,
    // Module-gated: the analytics module grants the analytics_module feature.
    entitlementKeys: [SURFACE_GATES.analytics.entitlement as string],
  },
];

const secondaryNavigation: NavigationItem[] = [
  {
    name: ui.layout.navKnowledgeBase,
    href: "/dashboard/knowledge",
    icon: BookOpen,
    // No permission required - everyone can access knowledge base.
  },
  {
    name: ui.layout.navSuppliers,
    href: "/dashboard/procurement",
    icon: Truck,
    // Procurement page enforces org:manage; keep the entry aligned.
    permission: SURFACE_GATES.suppliers.permission,
  },
  {
    name: ui.layout.navSchedules,
    href: "/dashboard/procurement/schedules",
    icon: CalendarClock,
    // Schedules page: same gating as the procurement section.
    permission: SURFACE_GATES.schedules.permission,
  },
];

// "Inteligencia Artificial" group. Only items with a real route are rendered:
// Copiloto IA → settings view=ai (org:manage). Entrenamiento/Automatizaciones
// have no product surface yet, so they are NOT linked unconditionally
// (design D2.2 / RBAC rule).
const aiNavigation: NavigationItem[] = [
  {
    name: ui.layout.navCopilot,
    href: "/dashboard/settings?view=ai",
    icon: Sparkles,
    queryKey: "view",
    queryValue: "ai",
    permission: SURFACE_GATES.aiCopilot.permission,
  },
];

const accountNavigation: NavigationItem[] = [
  { name: ui.layout.navSettings, href: "/dashboard/settings", icon: Settings },
];

interface SidebarProps {
  permissions: ServerPermissions;
}

export function Sidebar({
  permissions,
}: SidebarProps) {
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const isCollapsed = useSidebarStore((state) => state.isCollapsed);
  const entitlement = useEntitlementQuery().data;
  const hamburgerRef = useRef<HTMLButtonElement>(null);

  const closeMobileMenu = () => setIsMobileMenuOpen(false);

  // Escape closes the mobile drawer and returns focus to the hamburger.
  useEffect(() => {
    if (!isMobileMenuOpen) return;

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setIsMobileMenuOpen(false);
        hamburgerRef.current?.focus();
      }
    };

    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [isMobileMenuOpen]);

  // Filter navigation items based on permissions
  const visibleNavigation = useMemo(() => {
    const visible = (items: NavigationItem[]) =>
      items.filter((item) => {
        // Entitlement-gated items: hide when the entitlement payload is loaded
        // and no feature key is enabled. While entitlements load, default to
        // visible.
        if (item.entitlementKeys) {
          if (!entitlement) return true;
          return item.entitlementKeys.some((key) => entitlement.funcionalidades?.[key] === true);
        }

        // If no permission required, always show
        if (!item.permission && !("anyPermissions" in item)) return true;

        // Check if item has multiple permissions (anyPermissions)
        if ("anyPermissions" in item && item.anyPermissions) {
          // User must have at least one of the specified permissions
          return item.anyPermissions.some((perm) =>
            permissions.permissions.includes(perm as any)
          );
        }

        // Check single permission
        if (item.permission) {
          return permissions.permissions.includes(item.permission as any);
        }

        return true;
      });

    return {
      main: visible(mainNavigation),
      secondary: visible(secondaryNavigation),
      ai: visible(aiNavigation),
      account: visible(accountNavigation),
    };
  }, [permissions.permissions, entitlement]);

  const canViewAiInsights = useMemo(
    () => permissions.permissions.includes("org:manage"),
    [permissions.permissions]
  );

  const isItemActive = (item: NavigationItem) => {
    if (item.queryKey && item.queryValue) {
      return (
        pathname === "/dashboard/settings" &&
        searchParams.get(item.queryKey) === item.queryValue
      );
    }
    if (item.href.startsWith("/dashboard/") && item.href !== "/dashboard") {
      return pathname.startsWith(item.href);
    }
    return pathname === item.href;
  };

  const renderItem = (item: NavigationItem) => {
    const isActive = isItemActive(item);
    return (
      <Link
        key={item.name}
        href={item.href}
        onClick={closeMobileMenu}
        title={isCollapsed ? item.name : undefined}
        className={cn(
          "relative flex items-center overflow-hidden rounded-xl px-3 py-2.5 text-sm font-medium transition-colors",
          isCollapsed && "lg:justify-center lg:px-5",
          isActive
            ? "bg-emerald-500/10 border border-emerald-500/30 text-emerald-400"
            : "border border-transparent text-slate-400 hover:bg-slate-800 hover:text-white"
        )}
        aria-label={item.name}
        aria-current={isActive ? "page" : undefined}
      >
        <item.icon
          className={cn(
            "h-4 w-4 flex-none",
            isActive ? "text-emerald-400" : "text-slate-400"
          )}
        />
        <span
          className={cn(
            "ml-3 whitespace-nowrap transition-[margin,max-width,opacity] duration-200 ease-linear",
            isCollapsed
              ? "lg:ml-0 lg:max-w-0 lg:opacity-0"
              : "lg:max-w-[160px] lg:opacity-100"
          )}
        >
          {item.name}
        </span>
      </Link>
    );
  };

  return (
    <>
      {/* Mobile menu button */}
      <div className="fixed left-4 top-4 z-50 lg:hidden">
        <Button
          ref={hamburgerRef}
          variant="outline"
          size="icon"
          onClick={() => setIsMobileMenuOpen((open) => !open)}
          className="bg-background"
          aria-label={isMobileMenuOpen ? ui.layout.closeSidebar : ui.layout.openSidebar}
        >
          {isMobileMenuOpen ? (
            <X className="h-4 w-4" />
          ) : (
            <Menu className="h-4 w-4" />
          )}
        </Button>
      </div>

      {/* Mobile sidebar overlay */}
      {isMobileMenuOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 lg:hidden"
          onClick={closeMobileMenu}
        />
      )}

      {/* Sidebar */}
      <div
        className={cn(
          "fixed left-0 top-0 z-50 flex h-full w-64 flex-col border-r border-slate-800 bg-slate-900 transition-[transform,width] duration-200 ease-in-out lg:translate-x-0",
          isMobileMenuOpen
            ? "translate-x-0"
            : "-translate-x-full lg:translate-x-0",
          isCollapsed && "lg:w-20",
          "overflow-hidden"
        )}
      >
        {/* Logo / brand */}
        <div className="flex items-center gap-2.5 border-b border-slate-800 px-5 py-5">
          <LogoMark className="h-9 w-9" />
          <div
            className={cn(
              "max-w-[200px] overflow-hidden transition-[max-width,opacity] duration-200 ease-linear",
              isCollapsed ? "lg:max-w-0 lg:opacity-0" : "lg:opacity-100"
            )}
          >
            <div className="text-lg font-semibold tracking-tight text-white">
              {PRODUCT_NAME}
            </div>
          </div>
        </div>

        {/* Navigation */}
        <nav className="flex-1 overflow-y-auto p-4">
          <div className="space-y-1">
            {visibleNavigation.main.map(renderItem)}
          </div>

          <div className="mt-5 space-y-1 border-t border-slate-800 pt-4">
            {visibleNavigation.secondary.map(renderItem)}
          </div>

          {visibleNavigation.ai.length > 0 && (
            <div className="mt-5 space-y-1 border-t border-slate-800 pt-4">
              <p
                className={cn(
                  "px-3 pb-1.5 text-xs font-semibold uppercase tracking-wider text-slate-500",
                  isCollapsed && "lg:hidden"
                )}
              >
                {ui.layout.navAiGroup}
              </p>
              {visibleNavigation.ai.map(renderItem)}
            </div>
          )}

          {/* IA Insights card */}
          {canViewAiInsights && (
            <div
              className={cn(
                "mt-5 rounded-xl border border-emerald-500/20 bg-emerald-500/10 p-4",
                isCollapsed && "lg:hidden"
              )}
            >
              <div className="flex items-center gap-2">
                <GraduationCap className="h-5 w-5 text-emerald-400" aria-hidden />
                <span className="text-sm font-semibold text-white">
                  {ui.layout.aiInsightsTitle}
                </span>
              </div>
              <p className="mt-1.5 text-xs leading-relaxed text-slate-400">
                {ui.layout.aiInsightsBody}
              </p>
              <Link
                href="/dashboard/settings?view=ai"
                onClick={closeMobileMenu}
                className="mt-3 block w-full rounded-lg bg-emerald-500 py-2 text-center text-xs font-semibold text-white transition-colors hover:bg-emerald-600"
              >
                {ui.layout.aiInsightsCta}
              </Link>
            </div>
          )}
        </nav>

        {/* Account section */}
        <div
          className={cn(
            "border-t border-slate-800 px-4 py-4 transition-[padding] duration-200",
            isCollapsed && "lg:px-3"
          )}
        >
          <div className="space-y-2.5">
            {visibleNavigation.account.map(renderItem)}
          </div>
        </div>
      </div>
    </>
  );
}
