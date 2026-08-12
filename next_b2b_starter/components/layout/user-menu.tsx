"use client";

import Link from "next/link";
import { useMemo, useCallback, useTransition } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useTheme } from "next-themes";
import { Check, Monitor, Moon, Sun } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { buildLoginUrl } from "@/lib/auth/stytch-client";
import { useStytchConfig } from "@/lib/contexts/stytch-config-context";
import { usePermissions } from "@/lib/hooks/use-permissions";
import { useAuthContext } from "@/lib/contexts/auth-context";
import { resetCachedToken } from "@/lib/api/api/client/api-client";
import { logout } from "@/lib/actions/auth/logout";
import { ui } from "@/lib/copy/ui";

function getInitials(name?: string) {
  if (!name) return "?";
  const parts = name.trim().split(/\s+/);
  const first = parts[0]?.[0] || "";
  const last = parts.length > 1 ? parts[parts.length - 1][0] : "";
  return (first + last).toUpperCase();
}

export function UserMenu() {
  const { profile, isInitialized } = usePermissions();
  const authContext = useAuthContext();
  const queryClient = useQueryClient();
  const stytchConfig = useStytchConfig();
  const { theme, setTheme } = useTheme();
  const [isPending, startTransition] = useTransition();

  const loginHref = useMemo(() => {
    return buildLoginUrl(stytchConfig);
  }, [stytchConfig]);

  const handleLogout = useCallback(() => {
    startTransition(async () => {
      // Clear all client-side state
      authContext?.clearAuthState();
      queryClient.clear();
      resetCachedToken();

      // Call Server Action (will redirect to home page)
      await logout("/");
    });
  }, [authContext, queryClient]);

  if (!isInitialized) {
    return (
      <div className="h-9 w-24 animate-pulse rounded-md bg-muted" aria-label="Loading user" />
    );
  }

  if (!profile) {
    return (
      <Button asChild variant="default" className="h-9 bg-emerald-500 hover:bg-emerald-600 text-white">
        <a href={loginHref}>{ui.layout.login}</a>
      </Button>
    );
  }

  const display = profile.name || profile.email || "Cuenta";
  const initials = getInitials(profile.name || profile.email);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" className="h-9 gap-2 border-slate-700 bg-slate-800 text-slate-200 hover:bg-slate-700 hover:text-white">
          <span className="flex h-6 w-6 items-center justify-center rounded-full bg-emerald-500 text-xs font-semibold text-white">
            {initials}
          </span>
          <span className="hidden max-w-[160px] truncate text-sm md:inline">{display}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-52">
        <DropdownMenuItem asChild>
          <Link href="/dashboard">{ui.layout.navDashboard}</Link>
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuLabel className="text-xs text-muted-foreground">
          {ui.layout.theme}
        </DropdownMenuLabel>
        <DropdownMenuGroup>
          <DropdownMenuItem
            onClick={() => setTheme("light")}
            className="justify-between"
          >
            <span className="inline-flex items-center gap-2">
              <Sun className="h-4 w-4" aria-hidden />
              {ui.layout.themeLight}
            </span>
            {theme === "light" && <Check className="h-4 w-4" aria-hidden />}
          </DropdownMenuItem>
          <DropdownMenuItem
            onClick={() => setTheme("dark")}
            className="justify-between"
          >
            <span className="inline-flex items-center gap-2">
              <Moon className="h-4 w-4" aria-hidden />
              {ui.layout.themeDark}
            </span>
            {theme === "dark" && <Check className="h-4 w-4" aria-hidden />}
          </DropdownMenuItem>
          <DropdownMenuItem
            onClick={() => setTheme("system")}
            className="justify-between"
          >
            <span className="inline-flex items-center gap-2">
              <Monitor className="h-4 w-4" aria-hidden />
              {ui.layout.themeSystem}
            </span>
            {theme === "system" && <Check className="h-4 w-4" aria-hidden />}
          </DropdownMenuItem>
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={handleLogout} disabled={isPending}>
          {isPending ? ui.layout.logoutPending : ui.layout.logout}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
