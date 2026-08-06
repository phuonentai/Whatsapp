/**
 * React Hook for Permission Checking
 * Integrates with Stytch B2B session to provide permission-based access control
 */

'use client';

import { useMemo } from 'react';
import { useStytchMember } from '@stytch/nextjs/b2b';
import {
  hasRole,
  hasAnyRole,
  hasAllRoles,
} from '@/lib/auth/permission-utils';
import type { Permission } from '@/lib/auth/permissions';
import type { ProfileResponseDto } from '@/lib/api/api/dto/profile.dto';
import { useAuthContext } from '@/lib/contexts/auth-context';

export interface UsePermissionsReturn {
  /**
   * User profile details supplied by backend
   */
  profile: ProfileResponseDto | null;

  /**
   * Array of role names from Stytch session
   */
  roles: string[];

  /**
   * Array of all permissions granted to the user
   */
  permissions: string[];

  /**
   * Check if user has a specific permission
   * Supports wildcard matching (e.g., "invoice:*")
   */
  hasPermission: (permission: string) => boolean;

  /**
   * Check if user has ANY of the specified permissions (OR logic)
   */
  hasAnyPermission: (permissions: string[]) => boolean;

  /**
   * Check if user has ALL of the specified permissions (AND logic)
   */
  hasAllPermissions: (permissions: string[]) => boolean;

  /**
   * Check if user has a specific role
   */
  hasRole: (role: string) => boolean;

  /**
   * Check if user has ANY of the specified roles
   */
  hasAnyRole: (roles: string[]) => boolean;

  /**
   * Check if user has ALL of the specified roles
   */
  hasAllRoles: (roles: string[]) => boolean;

  /**
   * Whether the Stytch member is initialized
   */
  isInitialized: boolean;

  /**
   * Whether the user is authenticated
   */
  isAuthenticated: boolean;

  /**
   * Update cached auth state (used when profile changes client-side)
   */
  updateAuthState: (state: {
    profile?: ProfileResponseDto | null;
    roles?: string[];
    permissions?: Permission[];
  }) => void;
}

/**
 * Hook to access and check user permissions based on Stytch roles
 *
 * @returns Permission checking utilities and user state
 *
 * @example
 * function MyComponent() {
 *   const { hasPermission, hasAnyPermission, isAuthenticated } = usePermissions();
 *
 *   if (!isAuthenticated) return <Login />;
 *
 *   return (
 *     <>
 *       {hasPermission('invoice:create') && <CreateButton />}
 *       {hasAnyPermission(['invoice:view', 'invoice:*']) && <InvoiceList />}
 *     </>
 *   );
 * }
 */
export function usePermissions(): UsePermissionsReturn {
  const context = useAuthContext();

  // Safely get member, handling cases where provider might not be available
  let member, isInitialized;
  try {
    const stytchData = useStytchMember();
    member = stytchData.member;
    isInitialized = stytchData.isInitialized;
  } catch (error) {
    // If Stytch provider is not available, return empty state
    member = null;
    isInitialized = false;
  }

  // Extract roles from Stytch member
  // member.roles can be an array of strings (slugs) or objects
  const roles = useMemo(() => {
    const rawRoles = member?.roles;

    if (!rawRoles) return [];

    if (Array.isArray(rawRoles)) {
      return rawRoles
        .map(role => {
          if (typeof role === 'string') {
            return role;
          }

          if (role && typeof role === 'object') {
            return (
              (role as any).role_id ??
              (role as any).slug ??
              (role as any).name ??
              (role as any).id
            );
          }

          return undefined;
        })
        .filter((role): role is string => typeof role === 'string');
    }

    if (typeof rawRoles === 'string') {
      return [rawRoles];
    }

    return [];
  }, [member]);

  // Note: This hook is deprecated - use server-side permissions instead
  // Permissions should be computed on server and passed down as props
  const permissions = useMemo(() => {
    return [] as Permission[];
  }, []);

  // Create memoized permission check functions
  // Note: These won't work correctly without server-computed permissions
  const permissionChecks = useMemo(() => ({
    hasPermission: (_permission: string) => false,
    hasAnyPermission: (_perms: string[]) => false,
    hasAllPermissions: (_perms: string[]) => false,
    hasRole: (role: string) => hasRole(roles, role),
    hasAnyRole: (rolesToCheck: string[]) => hasAnyRole(roles, rolesToCheck),
    hasAllRoles: (rolesToCheck: string[]) => hasAllRoles(roles, rolesToCheck),
  }), [roles]);

  if (context) {
    return {
      profile: context.profile,
      roles: context.roles,
      permissions: context.permissions,
      hasPermission: context.hasPermission,
      hasAnyPermission: context.hasAnyPermission,
      hasAllPermissions: context.hasAllPermissions,
      hasRole: context.hasRole,
      hasAnyRole: context.hasAnyRole,
      hasAllRoles: context.hasAllRoles,
      isInitialized: context.isInitialized,
      isAuthenticated: context.isAuthenticated,
      updateAuthState: context.updateAuthState,
    };
  }

  return {
    profile: null,
    roles,
    permissions,
    ...permissionChecks,
    isInitialized,
    isAuthenticated: !!member && isInitialized,
    updateAuthState: () => undefined,
  };
}
