/**
 * Permission System
 * Based on Stytch RBAC - permissions are computed by backend
 * Frontend receives expanded permissions array from /auth/profile/me
 */

// Permission constants (actual Stytch RBAC permissions only)
export const PERMISSIONS = {
  // Resource Management
  RESOURCE_VIEW: "resource:view",
  RESOURCE_CREATE: "resource:create",
  RESOURCE_EDIT: "resource:edit",
  RESOURCE_DELETE: "resource:delete",
  RESOURCE_APPROVE: "resource:approve",

  // Organization Management
  ORG_VIEW: "org:view",
  ORG_MANAGE: "org:manage",

  // Invoice Management
  INVOICE_VIEW: "invoice:view",
  INVOICE_CREATE: "invoice:create",
  INVOICE_UPLOAD: "invoice:upload",
  INVOICE_DELETE: "invoice:delete",

  // Approvals
  APPROVALS_VIEW: "approvals:view",
  APPROVALS_APPROVE: "approvals:approve",

  // Duplicates
  DUPLICATES_VIEW: "duplicates:view",
  DUPLICATES_RESOLVE: "duplicates:resolve",

  // Payment Optimization
  PAYMENT_OPTIMIZATION_SCHEDULE: "payment:schedule",
  PAYMENT_OPTIMIZATION_EXPORT: "payment:export",
  PAYMENT_OPTIMIZATION_EXECUTE: "payment:execute",

  // Audit
  AUDIT_VIEW: "audit:view",
} as const;

export type Permission = (typeof PERMISSIONS)[keyof typeof PERMISSIONS];

/**
 * Permission resource and action groups for easier filtering
 */
export const PERMISSION_GROUPS = {
  RESOURCE: [
    PERMISSIONS.RESOURCE_VIEW,
    PERMISSIONS.RESOURCE_CREATE,
    PERMISSIONS.RESOURCE_EDIT,
    PERMISSIONS.RESOURCE_DELETE,
    PERMISSIONS.RESOURCE_APPROVE,
  ],
  ORGANIZATION: [PERMISSIONS.ORG_VIEW, PERMISSIONS.ORG_MANAGE],
  INVOICES: [
    PERMISSIONS.INVOICE_VIEW,
    PERMISSIONS.INVOICE_CREATE,
    PERMISSIONS.INVOICE_UPLOAD,
    PERMISSIONS.INVOICE_DELETE,
  ],
  APPROVALS: [PERMISSIONS.APPROVALS_VIEW, PERMISSIONS.APPROVALS_APPROVE],
  DUPLICATES: [PERMISSIONS.DUPLICATES_VIEW, PERMISSIONS.DUPLICATES_RESOLVE],
  PAYMENTS: [
    PERMISSIONS.PAYMENT_OPTIMIZATION_SCHEDULE,
    PERMISSIONS.PAYMENT_OPTIMIZATION_EXPORT,
    PERMISSIONS.PAYMENT_OPTIMIZATION_EXECUTE,
  ],
  AUDIT: [PERMISSIONS.AUDIT_VIEW],
} as const;
