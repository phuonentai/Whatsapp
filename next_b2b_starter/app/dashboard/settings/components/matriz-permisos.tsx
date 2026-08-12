"use client";

import { useMemo, useState } from "react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { RefreshCcw, AlertTriangle } from "lucide-react";
import type { RbacRole, RbacPermission } from "@/lib/api/api/repositories/rbac-repository";
import { ui } from "@/lib/copy/ui";

type CellState = "granted" | "partial" | "denied" | "broad";

interface ResourceRow {
  resource: string;
  actions: string[];
  cells: {
    roleId: string;
    state: CellState;
    perms: RbacPermission[];
    /** Literal wildcard (resource not defined in the policy): "permiso amplio". */
    broadWildcard?: string;
  }[];
}

interface MatrizPermisosProps {
  roles: RbacRole[];
  isLoading: boolean;
  isError: boolean;
  isRefetching?: boolean;
  onRetry: () => void;
}

function cellText(state: CellState): string {
  switch (state) {
    case "granted":
      return "✓";
    case "broad":
      return "✓*";
    case "partial":
      return "parcial";
    case "denied":
      return "—";
  }
}

export function MatrizPermisos({
  roles,
  isLoading,
  isError,
  isRefetching = false,
  onRetry,
}: MatrizPermisosProps) {
  const [filter, setFilter] = useState("");

  const rows = useMemo<ResourceRow[]>(() => {
    // Group permissions by resource across all roles; build the union of
    // actions per resource from the policy data (the DTO's per-role grants).
    const resources = new Map<
      string,
      { actions: Set<string>; cells: Map<string, { perms: RbacPermission[]; wildcard: boolean }> }
    >();

    for (const role of roles) {
      for (const perm of role.permissions ?? []) {
        const resource = perm.resource;
        if (!resource) continue;

        let entry = resources.get(resource);
        if (!entry) {
          entry = { actions: new Set(), cells: new Map() };
          resources.set(resource, entry);
        }
        if (perm.action === "*") {
          // Literal wildcard: resource not defined in the policy → broad.
          const cell = entry.cells.get(role.id) ?? { perms: [], wildcard: false };
          cell.wildcard = true;
          entry.cells.set(role.id, cell);
          continue;
        }
        if (perm.action) {
          entry.actions.add(perm.action);
        }
        const cell = entry.cells.get(role.id) ?? { perms: [], wildcard: false };
        cell.perms.push(perm);
        entry.cells.set(role.id, cell);
      }
    }

    const result: ResourceRow[] = [];
    for (const [resource, entry] of resources.entries()) {
      const actions = [...entry.actions].sort();
      result.push({
        resource,
        actions,
        cells: roles.map((role) => {
          const cell = entry.cells.get(role.id);
          if (!cell) {
            return { roleId: role.id, state: "denied", perms: [] };
          }
          if (cell.wildcard) {
            return {
              roleId: role.id,
              state: "broad",
              perms: cell.perms,
              broadWildcard: `${resource}:*`,
            };
          }
          const hasAll = actions.length > 0 && actions.every((a) =>
            cell.perms.some((p) => p.action === a)
          );
          const hasSome = cell.perms.length > 0;
          return {
            roleId: role.id,
            state: hasAll ? "granted" : hasSome ? "partial" : "denied",
            perms: cell.perms,
          };
        }),
      });
    }

    // Deterministic order: admin-relevant resources first, then alphabetical.
    result.sort((a, b) => a.resource.localeCompare(b.resource));
    return result;
  }, [roles]);

  const filteredRows = useMemo(() => {
    const q = filter.trim().toLowerCase();
    if (!q) return rows;
    return rows.filter((row) => row.resource.toLowerCase().includes(q));
  }, [rows, filter]);

  // Task 2.4: an empty role list is "policy unavailable", NEVER an empty
  // permissions matrix. The backend returns `{roles: []}` when the Stytch
  // policy fetch fails (breaker open / API down / empty cache).
  if (!isLoading && !isError && roles.length === 0) {
    return (
      <div className="flex min-h-[280px] flex-col items-center justify-center gap-4 rounded-xl border border-amber-200 bg-amber-50 px-6 py-12 text-center">
        <AlertTriangle className="h-8 w-8 text-amber-500" aria-hidden="true" />
        <div className="space-y-1">
          <p className="text-base font-semibold text-amber-900">
            {ui.teamPermissions.matrixEmpty}
          </p>
          <p className="max-w-md text-sm text-amber-800">
            {ui.teamPermissions.matrixEmptyBody}
          </p>
        </div>
        <Button variant="outline" onClick={onRetry} disabled={isRefetching}>
          <RefreshCcw
            className={`mr-2 h-4 w-4 ${isRefetching ? "animate-spin" : ""}`}
            aria-hidden="true"
          />
          {ui.teamPermissions.matrixEmptyRetry}
        </Button>
        <p className="text-xs text-amber-700">
          {ui.teamPermissions.matrixEmptyNeverEmpty}
        </p>
      </div>
    );
  }

  if (isError) {
    return (
      <div className="flex min-h-[280px] flex-col items-center justify-center gap-4 rounded-xl border border-red-200 bg-red-50 px-6 py-12 text-center">
        <AlertTriangle className="h-8 w-8 text-red-500" aria-hidden="true" />
        <p className="text-sm font-medium text-red-900">
          {ui.teamPermissions.matrixEmptyBody}
        </p>
        <Button variant="outline" onClick={onRetry} disabled={isRefetching}>
          <RefreshCcw
            className={`mr-2 h-4 w-4 ${isRefetching ? "animate-spin" : ""}`}
            aria-hidden="true"
          />
          {ui.teamPermissions.matrixEmptyRetry}
        </Button>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="space-y-4" aria-busy="true" aria-label={ui.teamPermissions.matrixTitle}>
        <Skeleton className="h-10 w-full max-w-xs" />
        <Skeleton className="h-[320px] w-full rounded-xl" />
      </div>
    );
  }

  // Admin column detection: pin it visually (sticky right) so it stays
  // visible when the matrix scrolls horizontally.
  const adminIndex = roles.findIndex((r) => r.id.toLowerCase() === "admin" || r.name.toLowerCase().includes("admin"));
  const hasAdminColumn = adminIndex >= 0;

  return (
    <TooltipProvider delayDuration={200}>
      <div className="space-y-4">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="w-full max-w-xs space-y-1">
            <Label htmlFor="matrix-filter" className="sr-only">
              {ui.teamPermissions.matrixFilterAria}
            </Label>
            <Input
              id="matrix-filter"
              type="search"
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              placeholder={ui.teamPermissions.matrixFilterPlaceholder}
              aria-label={ui.teamPermissions.matrixFilterAria}
              className="h-9"
            />
          </div>
          <div className="flex items-center gap-2">
            <span className="text-xs text-muted-foreground">
              {ui.teamPermissions.policySource}
            </span>
            <Button
              variant="outline"
              size="sm"
              onClick={onRetry}
              disabled={isRefetching}
            >
              <RefreshCcw
                className={`mr-1.5 h-3.5 w-3.5 ${isRefetching ? "animate-spin" : ""}`}
                aria-hidden="true"
              />
              {ui.teamPermissions.matrixRefresh}
            </Button>
          </div>
        </div>

        <div className="overflow-x-auto rounded-xl border border-gray-200 bg-white">
          <Table>
            <TableHeader>
              <TableRow className="border-gray-200 bg-gray-50/50">
                <TableHead
                  scope="col"
                  className="min-w-[140px] font-medium text-gray-700"
                >
                  Recurso
                </TableHead>
                {roles.map((role, index) => (
                  <TableHead
                    key={role.id}
                    scope="col"
                    className={
                      index === adminIndex
                        ? "sticky right-0 z-10 min-w-[110px] border-l border-gray-200 bg-gray-100 font-semibold text-gray-900"
                        : "min-w-[110px] font-medium text-gray-700"
                    }
                  >
                    {role.name || role.id}
                  </TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredRows.map((row) => (
                <TableRow key={row.resource} className="border-gray-100 hover:bg-gray-50/50">
                  <TableCell className="py-3 align-middle">
                    <p className="text-sm font-medium text-gray-900">{row.resource}</p>
                    {row.actions.length > 0 && (
                      <p className="text-xs text-gray-500">
                        {row.actions.join(", ")}
                      </p>
                    )}
                  </TableCell>
                  {row.cells.map((cell, index) => {
                    const role = roles.find((r) => r.id === cell.roleId);
                    const isAdmin = index === adminIndex;
                    return (
                      <TableCell
                        key={cell.roleId}
                        className={
                          isAdmin
                            ? "sticky right-0 z-10 border-l border-gray-200 bg-white px-3 py-3"
                            : "px-3 py-3"
                        }
                      >
                        <CellWithTooltip
                          cell={cell}
                          roleName={role?.name || role?.id || cell.roleId}
                          resource={row.resource}
                        />
                      </TableCell>
                    );
                  })}
                </TableRow>
              ))}
              {filteredRows.length === 0 && (
                <TableRow>
                  <TableCell
                    colSpan={roles.length + 1}
                    className="py-10 text-center text-sm text-muted-foreground"
                  >
                    {filter
                      ? `Sin recursos para “${filter}”.`
                      : ui.teamPermissions.matrixEmptyBody}
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>
      </div>
    </TooltipProvider>
  );
}

function CellWithTooltip({
  cell,
  roleName,
  resource,
}: {
  cell: ResourceRow["cells"][number];
  roleName: string;
  resource: string;
}) {
  const stateLabel =
    cell.state === "granted"
      ? ui.teamPermissions.matrixCellGranted
      : cell.state === "partial"
        ? ui.teamPermissions.matrixCellPartial
        : cell.state === "broad"
          ? ui.teamPermissions.matrixCellBroad
          : ui.teamPermissions.matrixCellDenied;

  const origin = cell.perms
    .map((p) => `${p.id}${p.displayName ? ` — ${p.displayName}` : ""}`)
    .join("\n");

  const tooltipBody = (() => {
    if (cell.state === "denied") {
      return `${roleName}: ${ui.teamPermissions.matrixCellDenied}`;
    }
    const parts: string[] = [];
    if (cell.broadWildcard) {
      parts.push(`${roleName}: ${cell.broadWildcard} (${ui.teamPermissions.matrixCellBroad})`);
    }
    if (cell.perms.length > 0) {
      parts.push(`${roleName} →`);
      parts.push(origin);
    }
    if (cell.state === "partial") {
      parts.push(`${ui.teamPermissions.matrixCellPartial} en ${resource}.`);
    }
    return parts.join("\n") || stateLabel;
  })();

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          aria-label={`${roleName} · ${resource}: ${stateLabel}`}
          className={`inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-sm font-medium transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-900/20 ${
            cell.state === "granted"
              ? "bg-emerald-50 text-emerald-700"
              : cell.state === "broad"
                ? "bg-emerald-50 text-emerald-700"
                : cell.state === "partial"
                  ? "bg-amber-50 text-amber-700"
                  : "bg-gray-50 text-gray-400"
          }`}
        >
          <span aria-hidden="true">{cellText(cell.state)}</span>
          <span className="sr-only">{stateLabel}</span>
        </button>
      </TooltipTrigger>
      <TooltipContent className="max-w-xs whitespace-pre-line">
        {tooltipBody}
      </TooltipContent>
    </Tooltip>
  );
}
