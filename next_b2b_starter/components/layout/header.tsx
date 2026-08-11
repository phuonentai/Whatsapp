// components/layout/header.tsx
"use client";

import Link from "next/link";
import { useMemo } from "react";
import {
  ChevronRight,
  LifeBuoy,
  PanelLeftClose,
  PanelLeftOpen,
  Search,
  Settings,
} from "lucide-react";
import { usePathname } from "next/navigation";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { useSidebarStore } from "@/lib/stores/sidebar-store";
import { useCommandPaletteStore } from "@/lib/stores/command-palette-store";
import { UserMenu } from "./user-menu";

export function Header() {
  const isSidebarCollapsed = useSidebarStore((state) => state.isCollapsed);
  const toggleSidebar = useSidebarStore((state) => state.toggle);
  const isAutoCollapsed = useSidebarStore((state) => state.isAutoCollapsed);
  const openPalette = useCommandPaletteStore((state) => state.openPalette);
  const pathname = usePathname();

  const breadcrumbItems = useMemo(() => {
    const segments = pathname.split("/").filter(Boolean);

    // Add Dashboard as first item if not already on dashboard
    const items: Array<{ label: string; href: string; isLast: boolean }> = [];

    if (segments.length > 0 && segments[0] !== 'dashboard') {
      items.push({
        label: 'Dashboard',
        href: '/dashboard',
        isLast: false,
      });
    }

    segments.forEach((segment, index) => {
      const href = `/${segments.slice(0, index + 1).join("/")}`;

      const label = segment
        .replace(/-/g, " ")
        .replace(/\b\w/g, (char) => char.toUpperCase());

      items.push({
        label: /^\d+$/.test(segment) ? `Invoice ${segment}` : label,
        href,
        isLast: index === segments.length - 1,
      });
    });

    return items;
  }, [pathname]);

  const pageTitle = breadcrumbItems[breadcrumbItems.length - 1]?.label ?? "Overview";

  return (
    <header className="sticky top-0 z-40 border-b border-border bg-background/90 backdrop-blur supports-[backdrop-filter]:bg-background/80">
      <div className="flex">
        <div
          className={cn(
            "hidden border-r border-border transition-[width] duration-200 lg:block",
            isSidebarCollapsed ? "w-20" : "w-64"
          )}
        />

        <div className="flex-1 px-6 py-4">
          <div className="flex flex-col gap-4">
            <div className="flex items-start justify-between gap-4">
              <div className="flex items-center gap-3">
                <Button
                  variant="outline"
                  size="icon"
                  onClick={toggleSidebar}
                  className="hidden h-9 w-9 lg:inline-flex"
                  aria-label={isSidebarCollapsed ? "Expand sidebar" : "Collapse sidebar"}
                  disabled={isAutoCollapsed}
                >
                  {isSidebarCollapsed ? (
                    <PanelLeftOpen className="h-4 w-4" />
                  ) : (
                    <PanelLeftClose className="h-4 w-4" />
                  )}
                </Button>

                <span className="hidden h-8 w-px bg-border lg:block" aria-hidden="true" />

                <div>
                  <h1 className="text-lg font-semibold text-foreground">{pageTitle}</h1>
                  <nav className="mt-1 flex flex-wrap items-center gap-1 text-sm text-muted-foreground">
                    {breadcrumbItems.map((item, index) => (
                      <span key={item.href} className="flex items-center gap-1">
                        {item.isLast ? (
                          <span className="font-medium text-foreground">{item.label}</span>
                        ) : (
                          <Link
                            href={item.href}
                            className="transition-colors hover:text-foreground"
                          >
                            {item.label}
                          </Link>
                        )}
                        {index < breadcrumbItems.length - 1 && (
                          <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
                        )}
                      </span>
                    ))}
                  </nav>
                </div>
              </div>

              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  className="hidden h-9 w-64 justify-start gap-2 text-sm font-normal text-muted-foreground sm:flex"
                  onClick={() => openPalette("search")}
                >
                  <Search className="h-4 w-4" aria-hidden />
                  <span className="flex-1 text-left">Search…</span>
                  <kbd className="pointer-events-none inline-flex h-5 select-none items-center gap-1 rounded border border-border bg-muted px-1.5 font-mono text-[10px] font-medium">
                    ⌘K
                  </kbd>
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-9 w-9 sm:hidden"
                  onClick={() => openPalette("search")}
                  aria-label="Search"
                >
                  <Search className="h-4 w-4" />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-9 w-9"
                  asChild
                >
                  <a
                    href={
                      process.env.NEXT_PUBLIC_CONTACT_EMAIL ||
                      "mailto:info@yourdomain.com"
                    }
                    aria-label="Support"
                  >
                    <LifeBuoy className="h-4 w-4" />
                  </a>
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-9 w-9"
                  asChild
                >
                  <Link href="/dashboard/settings" aria-label="Preferences">
                    <Settings className="h-4 w-4" />
                  </Link>
                </Button>
                <UserMenu />
              </div>
            </div>
          </div>
        </div>
      </div>
    </header>
  );
}
