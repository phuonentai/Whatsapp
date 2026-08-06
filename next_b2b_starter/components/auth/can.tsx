/**
 * Can Component - Permission-based conditional rendering
 * Wraps children and only renders them if the user has required permissions
 */

'use client';

import { ReactNode } from 'react';
import { usePermissions } from '@/lib/hooks/use-permissions';

export interface CanProps {
  /**
   * Single permission required to render children
   * @example "invoice:create"
   */
  permission?: string;

  /**
   * Multiple permissions - user must have ANY (OR logic)
   * @example ["invoice:view", "invoice:create"]
   */
  anyPermission?: string[];

  /**
   * Multiple permissions - user must have ALL (AND logic)
   * @example ["invoice:view", "approval:approve"]
   */
  allPermissions?: string[];

  /**
   * Single role required to render children
   * @example "admin"
   */
  role?: string;

  /**
   * Multiple roles - user must have ANY (OR logic)
   * @example ["admin", "manager"]
   */
  anyRole?: string[];

  /**
   * Multiple roles - user must have ALL (AND logic)
   * @example ["admin", "member"]
   */
  allRoles?: string[];

  /**
   * Render this when user doesn't have permission
   */
  fallback?: ReactNode;

  /**
   * Children to render when permission check passes
   */
  children: ReactNode;
}

/**
 * Permission wrapper component
 *
 * @example
 * // Single permission
 * <Can permission="invoice:create">
 *   <CreateButton />
 * </Can>
 *
 * @example
 * // Any of multiple permissions
 * <Can anyPermission={['invoice:view', 'invoice:*']}>
 *   <InvoiceList />
 * </Can>
 *
 * @example
 * // All permissions required
 * <Can allPermissions={['invoice:view', 'approval:approve']}>
 *   <ApprovalPanel />
 * </Can>
 *
 * @example
 * // With fallback
 * <Can permission="invoice:create" fallback={<AccessDenied />}>
 *   <CreateButton />
 * </Can>
 *
 * @example
 * // Role-based
 * <Can role="admin">
 *   <AdminPanel />
 * </Can>
 */
export function Can({
  permission,
  anyPermission,
  allPermissions,
  role,
  anyRole,
  allRoles,
  fallback = null,
  children,
}: CanProps) {
  const {
    hasPermission: checkPermission,
    hasAnyPermission: checkAnyPermission,
    hasAllPermissions: checkAllPermissions,
    hasRole: checkRole,
    hasAnyRole: checkAnyRole,
    hasAllRoles: checkAllRoles,
  } = usePermissions();

  // Check permission-based conditions
  if (permission && !checkPermission(permission)) {
    return <>{fallback}</>;
  }

  if (anyPermission && !checkAnyPermission(anyPermission)) {
    return <>{fallback}</>;
  }

  if (allPermissions && !checkAllPermissions(allPermissions)) {
    return <>{fallback}</>;
  }

  // Check role-based conditions
  if (role && !checkRole(role)) {
    return <>{fallback}</>;
  }

  if (anyRole && !checkAnyRole(anyRole)) {
    return <>{fallback}</>;
  }

  if (allRoles && !checkAllRoles(allRoles)) {
    return <>{fallback}</>;
  }

  // All checks passed, render children
  return <>{children}</>;
}
