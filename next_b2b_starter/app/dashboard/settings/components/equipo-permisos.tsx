"use client";

import { useMemo, useState } from "react";
import { useRouter, useSearchParams, usePathname } from "next/navigation";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { UserPlus, X, Clock3 } from "lucide-react";

import { MemberList, type RoleOption } from "./member-list";
import { InviteMember } from "./invite-member";
import { MatrizPermisos } from "./matriz-permisos";
import { ModulosResumen } from "./modulos-resumen";
import { PreviewImpacto } from "./preview-impacto";
import { CambiosRecientes } from "./cambios-recientes";
import { AuthPolicySection } from "./auth-policy-section";

import { useProfileQuery } from "@/lib/hooks/queries/use-profile-query";
import { useMembersQuery } from "@/lib/hooks/queries/use-members-query";
import { useRbacRolesQuery } from "@/lib/hooks/queries/use-rbac-roles-query";
import { useInviteMember } from "@/lib/hooks/mutations/use-invite-member";
import { usePermissions } from "@/lib/hooks/use-permissions";
import { useEntitlementQuery } from "@/lib/hooks/use-entitlement";
import { PERMISSIONS } from "@/lib/auth/permissions";
import type { InviteMemberRequest, MemberRole } from "@/lib/models/member.model";
import { ui } from "@/lib/copy/ui";

type AccessTab = "members" | "matrix" | "modules";

const VALID_TABS: AccessTab[] = ["members", "matrix", "modules"];

function parseTabParam(raw: string | null): AccessTab {
  if (raw && (VALID_TABS as string[]).includes(raw)) {
    return raw as AccessTab;
  }
  return "members";
}

/**
 * Vista consolidada "Equipo y permisos" (`?view=access`) — three layers:
 * members (assignment, editable by admin), permission matrix (read-only,
 * served from the Stytch policy), modules (plan metadata, summary only).
 *
 * Tabs are URL-addressable via `?tab=members|matrix|modules` (deep-link,
 * refresh and back/forward all work). The matrix and the role selector share
 * the SAME query (`/rbac/roles`, Stytch policy = runtime SSOT); the policy is
 * never edited from this UI.
 */
export function EquipoPermisos() {
  const { hasPermission, isInitialized: permissionsReady } = usePermissions();
  const canManageMembers = hasPermission(PERMISSIONS.ORG_MANAGE);
  const canViewAudit = hasPermission(PERMISSIONS.AUDIT_VIEW);

  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const tabParam = searchParams.get("tab");
  const activeTab = parseTabParam(tabParam);

  const [isInviteModalOpen, setInviteModalOpen] = useState(false);
  const [showPropagationNote, setShowPropagationNote] = useState(false);

  // Sync the tab with the URL: when the param arrives (deep-link/refresh/back)
  // the Tabs value derives from it; when the user clicks a tab we push the
  // param to the URL keeping `view=access` intact.
  const handleTabChange = (tab: string) => {
    const params = new URLSearchParams(searchParams.toString());
    params.set("tab", tab);
    router.replace(`${pathname}?${params.toString()}`, { scroll: false });
  };

  const {
    data: profile,
    isLoading: isProfileLoading,
    error: profileError,
  } = useProfileQuery({ enabled: permissionsReady });

  const {
    data: membersData,
    isLoading: isMembersLoading,
    isFetching: isMembersFetching,
    error: membersError,
    refetch: refetchMembers,
  } = useMembersQuery({
    organizationId: profile?.organizationId,
    page: 1,
    pageSize: 50,
    enabled: canManageMembers && Boolean(profile?.organizationId),
  });

  const {
    data: roles,
    isLoading: isRolesLoading,
    isError: isRolesError,
    isRefetching: isRolesRefetching,
    refetch: refetchRoles,
  } = useRbacRolesQuery({ enabled: permissionsReady && canManageMembers });

  const { data: entitlement } = useEntitlementQuery();

  const inviteMemberMutation = useInviteMember();

  const members = membersData?.members ?? [];
  const organizationId = profile?.organizationId ?? "";
  const hasOrganization = Boolean(organizationId);
  const canInviteMembers = canManageMembers && hasOrganization;

  // Role selector options sourced from the SAME query as the matrix
  // (`/rbac/roles`), with Spanish copy fallback. The English hardcoded role
  // descriptions are NOT the primary source anymore.
  const roleOptions = useMemo<RoleOption[]>(() => {
    const known: MemberRole[] = ["admin", "approver", "member"];
    return known.map((id) => {
      const role = roles?.find((r) => r.id.toLowerCase() === id);
      const fallback = {
        admin: {
          name: ui.teamPermissions.roleFallbackAdmin,
          description: ui.teamPermissions.roleFallbackAdminDesc,
        },
        approver: {
          name: ui.teamPermissions.roleFallbackApprover,
          description: ui.teamPermissions.roleFallbackApproverDesc,
        },
        member: {
          name: ui.teamPermissions.roleFallbackMember,
          description: ui.teamPermissions.roleFallbackMemberDesc,
        },
      }[id];
      return {
        id,
        name: role?.name || fallback.name,
        description: role?.description || fallback.description,
      };
    });
  }, [roles]);

  const handleInvite = async (request: InviteMemberRequest) => {
    if (!profile?.organizationId) return;
    try {
      await inviteMemberMutation.mutateAsync({
        request,
        organizationId: profile.organizationId,
      });
      setInviteModalOpen(false);
    } catch {
      // error toast handled by mutation
    }
  };

  const handleRefreshMembers = () => {
    refetchMembers();
  };

  // 403 gate (defense in depth — the settings allowlist already blocks the
  // view; never render member or matrix data without org:manage).
  if (permissionsReady && !canManageMembers) {
    return (
      <Alert variant="destructive" className="border border-red-200 bg-red-50">
        <AlertTitle>{ui.settings.statusError}</AlertTitle>
        <AlertDescription>
          No tienes permisos para gestionar el equipo y los permisos.
        </AlertDescription>
      </Alert>
    );
  }

  if (isProfileLoading || !permissionsReady) {
    return (
      <div className="space-y-6" aria-busy="true">
        <Skeleton className="h-10 w-64" />
        <Skeleton className="h-10 w-full max-w-sm" />
        <Skeleton className="h-[400px] w-full rounded-xl" />
      </div>
    );
  }

  if (profileError || !profile) {
    return (
      <div className="flex min-h-[300px] flex-col items-center justify-center gap-4 rounded-xl border border-red-200 bg-red-50 px-6 py-12 text-center">
        <p className="text-sm font-medium text-red-900">
          {profileError?.message || ui.common.unexpectedError}
        </p>
        <Button variant="outline" onClick={() => refetchMembers()}>
          {ui.common.retry}
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-1 sm:flex-row sm:items-end sm:justify-between">
        <div className="space-y-1">
          <h2 className="text-xl font-semibold text-gray-900">
            {ui.teamPermissions.title}
          </h2>
          <p className="text-sm text-gray-600">{ui.teamPermissions.description}</p>
        </div>
        <span className="text-xs text-muted-foreground">
          {ui.teamPermissions.policySource}
        </span>
      </div>

      <Tabs value={activeTab} onValueChange={handleTabChange}>
        <TabsList className="w-full justify-start sm:w-auto">
          <TabsTrigger value="members">{ui.teamPermissions.tabMembers}</TabsTrigger>
          <TabsTrigger value="matrix">{ui.teamPermissions.tabMatrix}</TabsTrigger>
          <TabsTrigger value="modules">{ui.teamPermissions.tabModules}</TabsTrigger>
        </TabsList>

        <TabsContent value="members" className="space-y-6 pt-4">
          {/* JIT / auth-policy card (stytch-enterprise-suite): domain join,
              SSO-JIT and allowed methods. Rendered only for org:manage. */}
          <AuthPolicySection />

          {showPropagationNote && (
            <div
              role="status"
              className="flex items-center justify-between gap-3 rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3"
            >
              <div className="flex items-center gap-2 text-sm text-emerald-900">
                <Clock3 className="h-4 w-4 flex-none text-emerald-600" aria-hidden="true" />
                {ui.teamPermissions.changesApplyNote}
              </div>
              <button
                type="button"
                onClick={() => setShowPropagationNote(false)}
                aria-label="Cerrar aviso"
                className="rounded p-1 text-emerald-700 hover:bg-emerald-100"
              >
                <X className="h-4 w-4" aria-hidden="true" />
              </button>
            </div>
          )}

          <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="space-y-1">
              <h3 className="text-base font-semibold text-gray-900">
                {ui.teamPermissions.teamRosterTitle}
              </h3>
              <p className="text-sm text-gray-600">
                {members.length} {members.length === 1 ? "miembro" : "miembros"}
              </p>
            </div>
            <Button
              onClick={() => setInviteModalOpen(true)}
              disabled={!canInviteMembers}
              className="w-full bg-gray-900 text-white hover:bg-gray-800 sm:w-auto"
            >
              <UserPlus className="mr-2 h-4 w-4" aria-hidden="true" />
              {ui.teamPermissions.addMemberCta}
            </Button>
          </div>

          {membersError && (
            <Alert variant="destructive" className="border border-red-200 bg-red-50">
              <AlertTitle>{ui.common.unexpectedError}</AlertTitle>
              <AlertDescription>
                {membersError.message || "No se pudieron cargar los miembros"}
              </AlertDescription>
            </Alert>
          )}

          {isMembersLoading && members.length === 0 ? (
            <Skeleton className="h-[300px] w-full rounded-xl" />
          ) : members.length === 0 && !membersError ? (
            <div className="flex min-h-[220px] flex-col items-center justify-center gap-4 rounded-xl border border-gray-200 bg-gray-50 px-6 py-12 text-center">
              <p className="text-sm font-medium text-gray-900">
                {ui.teamPermissions.emptyMembersTitle}
              </p>
              <p className="text-sm text-gray-500">
                {ui.teamPermissions.emptyMembersBody}
              </p>
              <Button
                variant="outline"
                onClick={() => setInviteModalOpen(true)}
                disabled={!canInviteMembers}
              >
                <UserPlus className="mr-2 h-4 w-4" aria-hidden="true" />
                {ui.teamPermissions.inviteCta}
              </Button>
            </div>
          ) : (
            <MemberList
              members={members}
              canManage={canManageMembers}
              currentUserId={profile.id}
              organizationId={organizationId}
              isFetching={isMembersFetching}
              onMemberUpdate={handleRefreshMembers}
              roleOptions={roleOptions}
              onRoleChange={() => setShowPropagationNote(true)}
            />
          )}

          <PreviewImpacto
            members={members}
            roles={roles ?? []}
            entitlement={entitlement}
          />

          {canViewAudit && <CambiosRecientes />}
        </TabsContent>

        <TabsContent value="matrix" className="space-y-6 pt-4">
          <MatrizPermisos
            roles={roles ?? []}
            isLoading={isRolesLoading}
            isError={isRolesError}
            isRefetching={isRolesRefetching}
            onRetry={() => refetchRoles()}
          />
        </TabsContent>

        <TabsContent value="modules" className="space-y-6 pt-4">
          <ModulosResumen />
        </TabsContent>
      </Tabs>

      <Dialog
        open={isInviteModalOpen}
        onOpenChange={(open) => {
          if (inviteMemberMutation.isPending) return;
          setInviteModalOpen(open);
        }}
      >
        <DialogContent id="invite-member-dialog" className="sm:max-w-lg">
          <DialogHeader className="space-y-2 text-left">
            <DialogTitle className="text-xl font-semibold text-gray-900">
              {ui.teamPermissions.addMemberCta}
            </DialogTitle>
            <DialogDescription className="text-sm text-gray-600">
              Envía una invitación segura y asigna el acceso correcto.
            </DialogDescription>
          </DialogHeader>
          <div className="pt-4">
            <InviteMember
              canInvite={canInviteMembers}
              onInvite={handleInvite}
            />
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
