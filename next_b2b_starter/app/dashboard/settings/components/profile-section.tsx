"use client";

import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { UserProfile, MemberHelpers } from "@/lib/models/member.model";
import { usePermissions } from "@/lib/hooks/use-permissions";
import { PERMISSIONS } from "@/lib/auth/permissions";
import { queryKeys } from "@/lib/hooks/queries/query-keys";
import { organizationRepository } from "@/lib/api/api/repositories/organization-repository";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

interface ProfileSectionProps {
  profile: UserProfile;
}

export function ProfileSection({ profile }: ProfileSectionProps) {
  const { hasPermission, isInitialized } = usePermissions();
  const canManage = isInitialized && hasPermission(PERMISSIONS.ORG_MANAGE);

  const [isEditing, setIsEditing] = useState(false);
  const [workspaceName, setWorkspaceName] = useState(
    profile.organizationName || ""
  );
  const [isSaving, setIsSaving] = useState(false);
  const [validationError, setValidationError] = useState<string | null>(null);
  const queryClient = useQueryClient();

  const roleConfig = MemberHelpers.getRoleConfig(profile.role);
  const displayName =
    profile.name?.trim() ||
    (profile.email ? profile.email.split("@")[0] : "AP Cash member");

  const handleSaveWorkspace = async () => {
    const name = workspaceName.trim();
    if (!name) {
      setValidationError("El nombre del workspace no puede estar vacío.");
      return;
    }
    setValidationError(null);
    setIsSaving(true);
    try {
      await organizationRepository.updateOrganization({
        name,
        status: profile.organizationStatus || "active",
      });
      await queryClient.invalidateQueries({
        queryKey: queryKeys.profile.all,
      });
      setIsEditing(false);
      toast.success("Workspace actualizado");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "No se pudo actualizar el workspace"
      );
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="grid gap-6 lg:grid-cols-2">
      <section className="rounded-2xl border border-gray-200 bg-white p-6 shadow-sm">
        <header className="space-y-1">
          <p className="text-xs font-semibold uppercase tracking-[0.2em] text-gray-500">
            Account owner
          </p>
          <h3 className="text-2xl font-semibold text-gray-900">{displayName}</h3>
          <p className="text-sm text-gray-600">
            These details identify you across automations and approvals.
          </p>
        </header>

        <dl className="mt-8 space-y-5 text-sm">
          <div className="flex flex-col gap-1.5 sm:flex-row sm:items-center sm:justify-between">
            <dt className="text-xs font-semibold uppercase tracking-wide text-gray-500">
              Email
            </dt>
            <dd className="text-base font-medium text-gray-900">{profile.email}</dd>
          </div>

          <div className="flex flex-col gap-1.5 sm:flex-row sm:items-center sm:justify-between">
            <dt className="text-xs font-semibold uppercase tracking-wide text-gray-500">
              Display name
            </dt>
            <dd className="text-base font-medium text-gray-900">
              {displayName}
            </dd>
          </div>

          <div className="flex flex-col gap-2">
            <dt className="text-xs font-semibold uppercase tracking-wide text-gray-500">
              Access level
            </dt>
            <dd>
              <span
                className={`inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs font-medium ${roleConfig.color}`}
              >
                {roleConfig.label}
              </span>
              <p className="mt-2 text-xs text-gray-500">{roleConfig.description}</p>
            </dd>
          </div>
        </dl>
      </section>

      <section className="rounded-2xl border border-gray-200 bg-white p-6 shadow-sm">
        <header className="space-y-1">
          <p className="text-xs font-semibold uppercase tracking-[0.2em] text-gray-500">
            Workspace
          </p>
          <div className="flex items-center justify-between gap-4">
            <h3 className="text-xl font-semibold text-gray-900">
              {profile.organizationName || "No workspace connected"}
            </h3>
            {canManage && profile.organizationName && !isEditing && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => {
                  setWorkspaceName(profile.organizationName);
                  setIsEditing(true);
                }}
              >
                Editar
              </Button>
            )}
          </div>
          <p className="text-sm text-gray-600">
            Configure branding, invite collaborators, and manage approvals within this workspace.
          </p>
        </header>

        {isEditing && (
          <div className="mt-4 space-y-2">
            <Label htmlFor="workspace-name" className="text-sm font-medium">
              Nombre del workspace
            </Label>
            <Input
              id="workspace-name"
              type="text"
              value={workspaceName}
              onChange={(e) => setWorkspaceName(e.target.value)}
              disabled={isSaving}
            />
            {validationError && (
              <p className="text-xs text-red-600">{validationError}</p>
            )}
            <div className="flex items-center gap-2">
              <Button
                size="sm"
                onClick={handleSaveWorkspace}
                disabled={isSaving}
                className="bg-gray-900 text-white hover:bg-gray-800"
              >
                {isSaving ? "Guardando…" : "Guardar"}
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  setIsEditing(false);
                  setValidationError(null);
                  setWorkspaceName(profile.organizationName || "");
                }}
                disabled={isSaving}
              >
                Cancelar
              </Button>
            </div>
          </div>
        )}

        <div className="mt-8 space-y-4 text-sm">
          <div className="rounded-xl border border-dashed border-gray-200 bg-gray-50 px-4 py-3">
            <p className="text-xs font-semibold uppercase tracking-wide text-gray-500">
              Workspace ID
            </p>
            <p className="mt-1 font-medium text-gray-900">
              {profile.organizationId || "Not assigned"}
            </p>
            <p className="mt-2 text-xs text-gray-500">
              You&apos;ll need this ID when connecting AP Cash to external approval tools.
            </p>
          </div>
          <p className="text-xs text-gray-500">
            Need to switch workspaces or update billing ownership? Reach out to support so we can
            take care of it for you.
          </p>
        </div>
      </section>
    </div>
  );
}
