/**
 * Server-Side Permission Utilities
 * Fetch permissions from backend API with proper cache control
 */

import type { B2BSessionsAuthenticateResponse } from "stytch";

import { profileRepository } from "@/lib/api/api/repositories/profile-repository";
import type { ProfileResponseDto } from "@/lib/api/api/dto/profile.dto";
import { PERMISSIONS } from "./permissions";
import type { Permission } from "./permissions";

export interface ServerPermissions {
  profile: ProfileResponseDto | null;
  roles: string[];
  permissions: string[];
  canViewInvoices: boolean;
  canCreateInvoices: boolean;
  canUploadInvoices: boolean;
  canDeleteInvoices: boolean;
  canViewApprovals: boolean;
  canApproveInvoices: boolean;
  canViewDuplicates: boolean;
  canResolveDuplicates: boolean;
  canSchedulePayments: boolean;
  canExportPayments: boolean;
  canExecutePayments: boolean;
  canViewAudit: boolean;
  canViewOrganization: boolean;
  canManageOrganization: boolean;
  canManageSubscriptions: boolean; // Derived from org:manage
  backendAvailable: boolean;
  backendError?: string | null;
}

const KNOWN_PERMISSIONS = new Set<string>(Object.values(PERMISSIONS));

/**
 * Compute all permissions for the user based on their session
 * Fetches permissions from backend API - NO CACHING to ensure fresh data
 *
 * Architecture:
 * 1. Get email/name from session.member (Stytch session object)
 * 2. Fetch permissions from backend /auth/profile/me API
 * 3. Backend computes permissions from Stytch RBAC
 * 4. Backend validates permissions on every API call (security maintained)
 */
export async function getServerPermissions(
  session: B2BSessionsAuthenticateResponse | null
): Promise<ServerPermissions> {
  const emptyPermissions: ServerPermissions = {
    profile: null,
    roles: [],
    permissions: [],
    canViewInvoices: false,
    canCreateInvoices: false,
    canUploadInvoices: false,
    canDeleteInvoices: false,
    canViewApprovals: false,
    canApproveInvoices: false,
    canViewDuplicates: false,
    canResolveDuplicates: false,
    canSchedulePayments: false,
    canExportPayments: false,
    canExecutePayments: false,
    canViewAudit: false,
    canViewOrganization: false,
    canManageOrganization: false,
    canManageSubscriptions: false,
    backendAvailable: true,
    backendError: null,
  };

  if (!session || !session.session_jwt) {
    return emptyPermissions;
  }

  try {
    // Fetch profile with backend-computed permissions
    const profile = await profileRepository.getProfile(session.session_jwt);

    if (!profile) {
      return emptyPermissions;
    }

    // Defensive: Handle missing or undefined permissions/roles fields
    const rawPermissions = Array.isArray(profile.permissions)
      ? profile.permissions
      : [];
    const rawRoles = Array.isArray(profile.roles)
      ? profile.roles
      : [];

    // Filter to only valid string permissions (trust backend as source of truth)
    const permissions = rawPermissions.filter(
      (permission): permission is string =>
        typeof permission === 'string' && permission.trim().length > 0
    );
    const roles = rawRoles.filter(
      (role): role is string => typeof role === 'string'
    );

    return {
      profile,
      roles,
      permissions: permissions as string[],
      canViewInvoices: permissions.includes(PERMISSIONS.INVOICE_VIEW),
      canCreateInvoices: permissions.includes(PERMISSIONS.INVOICE_CREATE),
      canUploadInvoices: permissions.includes(PERMISSIONS.INVOICE_UPLOAD),
      canDeleteInvoices: permissions.includes(PERMISSIONS.INVOICE_DELETE),
      canViewApprovals: permissions.includes(PERMISSIONS.APPROVALS_VIEW),
      canApproveInvoices: permissions.includes(PERMISSIONS.APPROVALS_APPROVE),
      canViewDuplicates: permissions.includes(PERMISSIONS.DUPLICATES_VIEW),
      canResolveDuplicates: permissions.includes(
        PERMISSIONS.DUPLICATES_RESOLVE
      ),
      canSchedulePayments: permissions.includes(
        PERMISSIONS.PAYMENT_OPTIMIZATION_SCHEDULE
      ),
      canExportPayments: permissions.includes(
        PERMISSIONS.PAYMENT_OPTIMIZATION_EXPORT
      ),
      canExecutePayments: permissions.includes(
        PERMISSIONS.PAYMENT_OPTIMIZATION_EXECUTE
      ),
      canViewAudit: permissions.includes(PERMISSIONS.AUDIT_VIEW),
      canViewOrganization: permissions.includes(PERMISSIONS.ORG_VIEW),
      canManageOrganization: permissions.includes(PERMISSIONS.ORG_MANAGE),
      canManageSubscriptions: permissions.includes(PERMISSIONS.ORG_MANAGE),
      backendAvailable: true,
      backendError: null,
    };
  } catch (error) {
    const code = (error as any)?.cause?.code;
    const baseMessage = error instanceof Error ? error.message : "Unknown error";
    const errorMessage = code ? `${code}: ${baseMessage}` : baseMessage;

    console.error('[Auth] Failed to fetch profile from backend:', {
      message: baseMessage,
      code,
      error,
    });

    return {
      ...emptyPermissions,
      backendAvailable: false,
      backendError: errorMessage,
    };
  }
}
