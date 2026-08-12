"use client";

import Link from "next/link";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Boxes, CreditCard, SlidersHorizontal } from "lucide-react";
import { useModules } from "@/lib/hooks/use-entitlement";
import { useModulesCatalogQuery } from "@/lib/hooks/queries/use-modules-queries";
import { ui } from "@/lib/copy/ui";

/**
 * Módulos tab — summary of active modules with a plan-source badge and links
 * to the existing `?view=modules` (toggles) and `?view=subscription` (upgrade)
 * views. Deliberately NO duplicated toggles: module configuration stays in
 * `?view=modules` (single source).
 */
export function ModulosResumen() {
  const modules = useModules();
  const { data: catalog, isLoading } = useModulesCatalogQuery();

  const activeKeys = Object.entries(modules)
    .filter(([, state]) => state.enabled)
    .map(([key]) => key)
    .sort();

  const nameFor = (key: string): string =>
    catalog?.find((m) => m.key === key)?.name ?? key;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Boxes className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
          <h3 className="text-sm font-semibold text-gray-900">
            {ui.teamPermissions.moduleSummaryTitle}
          </h3>
        </div>
        <div className="flex items-center gap-2">
          <Link href="/dashboard/settings?view=subscription">
            <Button variant="outline" size="sm">
              <CreditCard className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
              {ui.teamPermissions.moduleSummarySubscriptionLink}
            </Button>
          </Link>
          <Link href="/dashboard/settings?view=modules">
            <Button variant="outline" size="sm">
              <SlidersHorizontal className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
              {ui.teamPermissions.moduleSummaryModulesLink}
            </Button>
          </Link>
        </div>
      </div>

      {isLoading ? (
        <div className="space-y-2">
          <Skeleton className="h-12 w-full rounded-xl" />
          <Skeleton className="h-12 w-full rounded-xl" />
        </div>
      ) : activeKeys.length === 0 ? (
        <div className="rounded-xl border border-gray-200 bg-gray-50 p-6 text-center text-sm text-gray-500">
          {ui.teamPermissions.moduleSummaryNone}
        </div>
      ) : (
        <ul className="divide-y divide-gray-100 overflow-hidden rounded-xl border border-gray-200 bg-white">
          {activeKeys.map((key) => (
            <li
              key={key}
              className="flex items-center justify-between px-4 py-3"
            >
              <span className="text-sm font-medium text-gray-900">
                {nameFor(key)}
              </span>
              <Badge
                variant="outline"
                className="border-emerald-200 bg-emerald-50 text-emerald-700"
                aria-label={ui.teamPermissions.moduleSummarySourceBadge}
              >
                {ui.teamPermissions.moduleSummarySourceBadge}
              </Badge>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
