/**
 * Query Keys Factory
 *
 * Centralized query keys for TanStack Query.
 * This ensures type safety, consistency, and easy invalidation.
 *
 * Pattern:
 * - `all`: Base key for the entire resource
 * - `lists()`: All lists of this resource
 * - `list(filters)`: Specific filtered list
 * - `details()`: All detail views
 * - `detail(id)`: Specific detail view
 */

// import type { InvoiceListFilter } from "@/lib/models/invoice-list.model";
// import type { ApprovalRequestListFilter } from "@/lib/models/approval-request.model";
// import type { PaymentOptimizationListFilter } from "@/lib/models/payment-optimization.model";
// import type { AuditListFilter } from "@/lib/models/audit-trail.model";
// import type { PerformanceAnalyticsDateRange } from "@/lib/models/performance-analytics.model";
import type { DocumentListFilter } from "@/lib/models/document.model";

// Temporary type placeholders until models are created
type InvoiceListFilter = any;
type ApprovalRequestListFilter = any;
type PaymentOptimizationListFilter = any;
type AuditListFilter = any;
type PerformanceAnalyticsDateRange = any;

export const queryKeys = {
  /**
   * Profile query keys
   */
  profile: {
    all: ["profile"] as const,
    detail: () => [...queryKeys.profile.all, "detail"] as const,
  },

  /**
   * Members query keys
   */
  members: {
    all: ["members"] as const,
    lists: () => [...queryKeys.members.all, "list"] as const,
    list: (filters: {
      organizationId?: string;
      page?: number;
      pageSize?: number;
    }) => [...queryKeys.members.lists(), filters] as const,
    detail: (memberId: string) =>
      [...queryKeys.members.all, "detail", memberId] as const,
  },

  /**
   * Subscription query keys
   */
  subscription: {
    all: ["subscription"] as const,
    status: () => [...queryKeys.subscription.all, "status"] as const,
  },

  /**
   * Products query keys (Polar billing plans)
   */
  products: {
    all: ["products"] as const,
    lists: () => [...queryKeys.products.all, "list"] as const,
    list: [...["products"], "list"] as const,
  },

  /**
   * Invoices query keys
   */
  invoices: {
    all: ["invoices"] as const,
    lists: () => [...queryKeys.invoices.all, "list"] as const,
    list: (filters: InvoiceListFilter) =>
      [...queryKeys.invoices.lists(), filters] as const,
    details: () => [...queryKeys.invoices.all, "detail"] as const,
    detail: (invoiceId: number) =>
      [...queryKeys.invoices.details(), invoiceId] as const,
    duplicates: () => [...queryKeys.invoices.all, "duplicates"] as const,
    duplicate: (invoiceId: number) =>
      [...queryKeys.invoices.duplicates(), invoiceId] as const,
  },

  /**
   * Approvals query keys
   */
  approvals: {
    all: ["approvals"] as const,
    lists: () => [...queryKeys.approvals.all, "list"] as const,
    list: (filters: ApprovalRequestListFilter) =>
      [...queryKeys.approvals.lists(), filters] as const,
    details: () => [...queryKeys.approvals.all, "detail"] as const,
    detail: (approvalId: number) =>
      [...queryKeys.approvals.details(), approvalId] as const,
  },

  /**
   * Payments query keys
   */
  payments: {
    all: ["payments"] as const,
    lists: () => [...queryKeys.payments.all, "list"] as const,
    list: (filters: PaymentOptimizationListFilter) =>
      [...queryKeys.payments.lists(), filters] as const,
    details: () => [...queryKeys.payments.all, "detail"] as const,
    detail: (paymentId: number) =>
      [...queryKeys.payments.details(), paymentId] as const,
  },

  /**
   * Audit trail query keys
   */
  audit: {
    all: ["audit"] as const,
    summaries: () => [...queryKeys.audit.all, "summaries"] as const,
    summaryList: (filters: AuditListFilter) =>
      [...queryKeys.audit.summaries(), filters] as const,
    timelines: () => [...queryKeys.audit.all, "timelines"] as const,
    timeline: (invoiceId: number) =>
      [...queryKeys.audit.timelines(), invoiceId] as const,
  },

  /**
   * Performance Analytics query keys
   */
  analytics: {
    all: ["analytics"] as const,
    roi: () => [...queryKeys.analytics.all, "roi"] as const,
    roiMetrics: (dateRange: PerformanceAnalyticsDateRange) =>
      [...queryKeys.analytics.roi(), dateRange] as const,
    breakdown: () => [...queryKeys.analytics.all, "breakdown"] as const,
    savingsBreakdown: (dateRange: PerformanceAnalyticsDateRange) =>
      [...queryKeys.analytics.breakdown(), dateRange] as const,
  },

  /**
   * Dashboard query keys
   */
  dashboard: {
    all: ["dashboard"] as const,
    data: () => [...queryKeys.dashboard.all, "data"] as const,
  },

  /**
   * Documents query keys
   */
  documents: {
    all: ["documents"] as const,
    lists: () => ["documents", "list"] as const,
    list: (filters: DocumentListFilter) =>
      ["documents", "list", filters] as const,
    details: () => ["documents", "detail"] as const,
    detail: (documentId: number) =>
      ["documents", "detail", documentId] as const,
  },

  /**
   * Cognitive / Chat query keys
   */
  cognitive: {
    all: ["cognitive"] as const,
    sessions: () => ["cognitive", "sessions"] as const,
    messages: (sessionId: number) =>
      ["cognitive", "messages", sessionId] as const,
  },
} as const;

/**
 * Helper function to invalidate all queries for a resource
 *
 * @example
 * ```ts
 * // Invalidate all profile queries
 * queryClient.invalidateQueries({ queryKey: queryKeys.profile.all });
 *
 * // Invalidate specific members list
 * queryClient.invalidateQueries({
 *   queryKey: queryKeys.members.list({ organizationId: 'org-123' })
 * });
 * ```
 */
