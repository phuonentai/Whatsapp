/**
 * PermissionGate Component - Page-level permission guard
 * Protects entire pages or sections with permission checks
 */

'use client';

import { ReactNode } from 'react';
import { useRouter } from 'next/navigation';
import { usePermissions } from '@/lib/hooks/use-permissions';
import { AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';

export interface PermissionGateProps {
  /**
   * Permissions required - user must have ALL (AND logic)
   * @example ["invoice:view"]
   */
  required?: string[];

  /**
   * Permissions required - user must have ANY (OR logic)
   * @example ["invoice:view", "invoice:*"]
   */
  anyRequired?: string[];

  /**
   * Roles required - user must have ALL (AND logic)
   * @example ["admin"]
   */
  requiredRoles?: string[];

  /**
   * Roles required - user must have ANY (OR logic)
   * @example ["admin", "manager"]
   */
  anyRequiredRole?: string[];

  /**
   * Custom fallback component when permission check fails
   */
  fallback?: ReactNode;

  /**
   * Show default access denied message
   * @default true
   */
  showAccessDenied?: boolean;

  /**
   * Redirect to this path when permission check fails
   * @example "/dashboard"
   */
  redirectTo?: string;

  /**
   * Children to render when permission check passes
   */
  children: ReactNode;
}

/**
 * Default access denied message
 */
function AccessDenied({ onGoBack }: { onGoBack: () => void }) {
  return (
    <div className="flex min-h-[400px] items-center justify-center p-8">
      <div className="w-full max-w-md">
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>Access Denied</AlertTitle>
          <AlertDescription className="mt-2">
            You don't have permission to access this page. Please contact your
            administrator if you believe this is a mistake.
          </AlertDescription>
        </Alert>
        <div className="mt-6 flex justify-center">
          <Button onClick={onGoBack} variant="outline">
            Go Back
          </Button>
        </div>
      </div>
    </div>
  );
}

/**
 * Page-level permission guard
 *
 * @example
 * // Require single permission
 * <PermissionGate required={['invoice:view']}>
 *   <InvoicePage />
 * </PermissionGate>
 *
 * @example
 * // Require any of multiple permissions
 * <PermissionGate anyRequired={['invoice:view', 'invoice:*']}>
 *   <InvoicePage />
 * </PermissionGate>
 *
 * @example
 * // Require multiple permissions (all must be present)
 * <PermissionGate required={['invoice:view', 'approval:approve']}>
 *   <ComplexPage />
 * </PermissionGate>
 *
 * @example
 * // With custom fallback
 * <PermissionGate
 *   required={['invoice:view']}
 *   fallback={<CustomAccessDenied />}
 * >
 *   <InvoicePage />
 * </PermissionGate>
 *
 * @example
 * // Redirect to another page
 * <PermissionGate
 *   required={['invoice:view']}
 *   redirectTo="/dashboard"
 * >
 *   <InvoicePage />
 * </PermissionGate>
 *
 * @example
 * // Role-based protection
 * <PermissionGate anyRequiredRole={['admin', 'manager']}>
 *   <AdminPanel />
 * </PermissionGate>
 */
export function PermissionGate({
  required,
  anyRequired,
  requiredRoles,
  anyRequiredRole,
  fallback,
  showAccessDenied = true,
  redirectTo,
  children,
}: PermissionGateProps) {
  const router = useRouter();
  const {
    hasAllPermissions,
    hasAnyPermission,
    hasAllRoles,
    hasAnyRole,
    isInitialized,
  } = usePermissions();

  // Wait for initialization
  if (!isInitialized) {
    return (
      <div className="flex min-h-[400px] items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-gray-900" />
      </div>
    );
  }

  // Check permissions
  let hasAccess = true;

  if (required && !hasAllPermissions(required)) {
    hasAccess = false;
  }

  if (anyRequired && !hasAnyPermission(anyRequired)) {
    hasAccess = false;
  }

  if (requiredRoles && !hasAllRoles(requiredRoles)) {
    hasAccess = false;
  }

  if (anyRequiredRole && !hasAnyRole(anyRequiredRole)) {
    hasAccess = false;
  }

  // Access granted
  if (hasAccess) {
    return <>{children}</>;
  }

  // Access denied - handle redirect
  if (redirectTo) {
    router.push(redirectTo);
    return null;
  }

  // Access denied - show custom fallback
  if (fallback) {
    return <>{fallback}</>;
  }

  // Access denied - show default message
  if (showAccessDenied) {
    return <AccessDenied onGoBack={() => router.back()} />;
  }

  // Don't show anything
  return null;
}
