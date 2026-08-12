"use client";

import Link from "next/link";
import { format } from "date-fns";
import { ScrollText } from "lucide-react";
import { useActivitiesQuery } from "@/lib/hooks/queries/use-crm-queries";
import { ui } from "@/lib/copy/ui";

/**
 * "Cambios recientes" — compact list of recent system events from the existing
 * audit ledger, plus a link to the full audit log.
 *
 * Gating contract: this component SHALL only be rendered when the user has
 * `org:manage` AND `audit:view` (same predicate as the audit-log view). The
 * parent (`EquipoPermisos`) enforces the gate, so without `audit:view` this
 * component is never mounted and no ledger request fires.
 */
export function CambiosRecientes() {
  const { data: activities, isLoading, error } = useActivitiesQuery({
    tipo: "sistema",
    limit: 5,
    offset: 0,
  });

  return (
    <section className="space-y-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <ScrollText className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
          <h3 className="text-sm font-semibold text-gray-900">
            {ui.teamPermissions.recentChangesTitle}
          </h3>
        </div>
        <Link
          href="/dashboard/settings?view=audit"
          className="text-sm font-medium text-emerald-700 hover:text-emerald-800 hover:underline"
        >
          {ui.teamPermissions.recentChangesLink}
        </Link>
      </div>

      {error ? (
        <p className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
          {error instanceof Error
            ? error.message
            : ui.common.unexpectedError}
        </p>
      ) : isLoading ? (
        <div className="space-y-2" aria-busy="true">
          <div className="h-12 animate-pulse rounded-lg border border-gray-100 bg-gray-50" />
          <div className="h-12 animate-pulse rounded-lg border border-gray-100 bg-gray-50" />
        </div>
      ) : !activities?.length ? (
        <p className="rounded-lg border border-gray-200 bg-gray-50 px-4 py-3 text-sm text-gray-500">
          {ui.teamPermissions.recentChangesEmpty}
        </p>
      ) : (
        <ul className="divide-y divide-gray-100 overflow-hidden rounded-lg border border-gray-200 bg-white">
          {activities.map((activity) => (
            <li
              key={activity.id}
              className="flex flex-col gap-1 px-4 py-3 sm:flex-row sm:items-start sm:justify-between"
            >
              <div className="min-w-0">
                <p className="truncate text-sm font-medium text-gray-900">
                  {activity.asunto || activity.contenido || ui.teamPermissions.recentChangesEmpty}
                </p>
                <p className="mt-0.5 text-xs text-gray-500">
                  {ui.teamPermissions.recentChangesActor.replace(
                    "{name}",
                    activity.realizada_por_nombre ||
                      ui.teamPermissions.recentChangesSystem
                  )}
                </p>
              </div>
              <p className="shrink-0 text-xs text-gray-400">
                {format(
                  new Date(activity.realizada_en),
                  "MMM d, yyyy 'a las' h:mm a"
                )}
              </p>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
