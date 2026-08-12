"use client";

import { useMemo, useRef, useState } from "react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import { Check, X, ChevronDown, Info } from "lucide-react";
import type { OrganizationMember } from "@/lib/models/member.model";
import type { RbacRole } from "@/lib/api/api/repositories/rbac-repository";
import {
  computeSurfaceAccess,
  type SurfaceKey,
  type SurfaceEntitlement,
} from "@/lib/auth/surface-gating";
import { ui } from "@/lib/copy/ui";

const SURFACE_LABELS: Record<SurfaceKey, string> = {
  inbox: ui.teamPermissions.previewSurfaceInbox,
  contacts: ui.teamPermissions.previewSurfaceContacts,
  aiCopilot: ui.teamPermissions.previewSurfaceAICopilot,
  knowledge: ui.teamPermissions.previewSurfaceKnowledge,
  invoices: ui.teamPermissions.previewSurfaceInvoices,
  payments: ui.teamPermissions.previewSurfacePayments,
  analytics: ui.teamPermissions.previewSurfaceAnalytics,
  suppliers: ui.teamPermissions.previewSurfaceSuppliers,
  schedules: ui.teamPermissions.previewSurfaceSchedules,
  settings: ui.teamPermissions.previewSurfaceSettings,
};

const SURFACE_ORDER: SurfaceKey[] = [
  "inbox",
  "contacts",
  "aiCopilot",
  "knowledge",
  "invoices",
  "payments",
  "analytics",
  "suppliers",
  "schedules",
  "settings",
];

interface PreviewImpactoProps {
  members: OrganizationMember[];
  /** Policy roles from `/rbac/roles` (role → permission definitions). */
  roles: RbacRole[];
  entitlement?: SurfaceEntitlement | null;
}

/**
 * Impact preview — "¿qué ve este miembro?". Derived from the SAME surface
 * gating used by the navigation (lib/auth/surface-gating.ts) plus the
 * member's role permissions from the Stytch policy, so it can never diverge
 * from the real application. Accessible disclosure (aria-expanded) with focus
 * moved inside when opened.
 */
export function PreviewImpacto({ members, roles, entitlement }: PreviewImpactoProps) {
  const [selectedId, setSelectedId] = useState<string>("");
  const [isOpen, setIsOpen] = useState(false);
  const regionRef = useRef<HTMLDivElement>(null);

  const selectedMember = useMemo(
    () => members.find((m) => m.id === selectedId) ?? null,
    [members, selectedId]
  );

  const access = useMemo(() => {
    if (!selectedMember) return null;
    const role = selectedMember.role;
    const roleDef = roles.find(
      (r) => r.id.toLowerCase() === role.toLowerCase()
    );
    const perms = (roleDef?.permissions ?? []).map((p) => p.id);
    return computeSurfaceAccess(perms, entitlement);
  }, [selectedMember, roles, entitlement]);

  const toggleOpen = (next: boolean) => {
    setIsOpen(next);
    if (next) {
      // Move focus inside the disclosure region once it is rendered.
      requestAnimationFrame(() => regionRef.current?.focus());
    }
  };

  const memberName = selectedMember?.name || selectedMember?.email || "";

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="preview-member" className="text-sm font-medium text-gray-900">
          {ui.teamPermissions.previewTitle}
        </Label>
        <Select
          value={selectedId}
          onValueChange={(value) => {
            setSelectedId(value);
          }}
        >
          <SelectTrigger
            id="preview-member"
            className="w-full justify-between text-left sm:w-80"
            aria-label={ui.teamPermissions.previewSelectAria}
          >
            <SelectValue placeholder={ui.teamPermissions.previewSelectMember} />
          </SelectTrigger>
          <SelectContent>
            {members.map((member) => (
              <SelectItem key={member.id} value={member.id}>
                {member.name || member.email}
                <span className="ml-2 text-xs text-muted-foreground">
                  {member.role}
                </span>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {selectedMember && access && (
        <div className="rounded-xl border border-gray-200 bg-white">
          <button
            type="button"
            onClick={() => toggleOpen(!isOpen)}
            aria-expanded={isOpen}
            aria-controls="preview-impacto-region"
            className="flex w-full items-center justify-between gap-3 rounded-xl px-4 py-3 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-900/20"
          >
            <span className="flex items-center gap-2">
              <span className="text-sm font-semibold text-gray-900">
                {memberName}
              </span>
              <span className="rounded-md bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600">
                {selectedMember.role}
              </span>
            </span>
            <ChevronDown
              className={`h-4 w-4 text-gray-400 transition-transform ${isOpen ? "rotate-180" : ""}`}
              aria-hidden="true"
            />
          </button>

          {isOpen && (
            <div
              id="preview-impacto-region"
              ref={regionRef}
              tabIndex={-1}
              className="border-t border-gray-100 px-4 py-3 focus:outline-none"
            >
              <ul className="grid grid-cols-1 gap-1.5 sm:grid-cols-2">
                {SURFACE_ORDER.map((surface) => {
                  const allowed = access[surface];
                  const label = SURFACE_LABELS[surface];
                  return (
                    <li
                      key={surface}
                      className="flex items-center gap-2 rounded-md px-2 py-1.5 text-sm"
                    >
                      {allowed ? (
                        <Check
                          className="h-4 w-4 flex-none text-emerald-600"
                          aria-hidden="true"
                        />
                      ) : (
                        <X
                          className="h-4 w-4 flex-none text-red-400"
                          aria-hidden="true"
                        />
                      )}
                      <span className={allowed ? "text-gray-900" : "text-gray-500"}>
                        {label}
                      </span>
                      <span className="sr-only">
                        {allowed
                          ? ui.teamPermissions.previewAccessible
                          : ui.teamPermissions.previewNotAccessible}
                      </span>
                    </li>
                  );
                })}
              </ul>
              <p className="mt-3 flex items-center gap-1.5 text-xs text-muted-foreground">
                <Info className="h-3.5 w-3.5" aria-hidden="true" />
                {ui.teamPermissions.previewBasedOnPolicy}
              </p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
