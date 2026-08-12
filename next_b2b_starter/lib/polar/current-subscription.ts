import { getMemberSession } from "@/lib/auth/stytch/server";
import { getServerPermissions } from "@/lib/auth/server-permissions";
import { getActiveSubscription } from "@/lib/polar/subscription";
import { getInvoiceUsage } from "@/lib/polar/usage";
import {
  isMercadoPagoEnabled,
  MERCADOPAGO_CHECKOUT_PLAN_ID,
} from "@/lib/mercadopago/config";

export interface SubscriptionSnapshot {
  id: string;
  status: string;
  currentPeriodStart: string;
  currentPeriodEnd: string | null;
  cancelAtPeriodEnd: boolean;
  customerId: string;
  productId: string;
  productName: string | null;
  productMetadata: Record<string, unknown> | null;
  trialEnd: string | null;
  // Additional Polar properties
  trialStart: string | null;
  recurringInterval: string;
  metadata: Record<string, unknown> | null;
  customFieldData: Record<string, unknown> | null;
  customerCancellationReason: string | null;
  customerCancellationComment: string | null;
}

export interface UsageSnapshot {
  meterId: string;
  customerId: string;
  included: number;
  used: number;
  remaining: number;
  periodStart: string;
  periodEnd: string;
}

export interface SubscriptionGateState {
  isAuthenticated: boolean;
  isActive: boolean;
  reason?: string;
  status?: string | null;
  productId: string | null;
  meterId: string | null;
  planId: string | null;
  subscription: SubscriptionSnapshot | null;
  usage: UsageSnapshot | null;
  backendAvailable: boolean;
  backendError?: string | null;
}

/**
 * Provider-agnostic subscription status from the backend
 * (GET /api/subscriptions/status). Both billing providers update it via
 * webhook/verify, so it is authoritative for MercadoPago organizations that
 * the Polar SDK path cannot see.
 */
export interface BackendSubscriptionStatus {
  organization_id?: number;
  external_id?: string;
  has_active_subscription: boolean;
  can_process_invoices?: boolean;
  invoice_count?: number;
  reason?: string;
  checked_at?: string;
}

function getGoBackendUrl(): string {
  return process.env.NEXT_PUBLIC_GO_BACKEND_URL ?? process.env.GO_BACKEND_URL ?? "http://localhost:8080";
}

/**
 * Fetches the backend subscription status using the session JWT (same bearer
 * pattern as the MP billing server actions). Returns null when the backend is
 * unreachable so callers degrade gracefully to the Polar result.
 */
export async function fetchBackendSubscriptionStatus(
  sessionJwt: string
): Promise<BackendSubscriptionStatus | null> {
  try {
    const response = await fetch(`${getGoBackendUrl()}/api/subscriptions/status`, {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${sessionJwt}`,
      },
      cache: "no-store",
    });

    if (!response.ok) {
      console.warn("[Polar] Backend subscription status unavailable", {
        status: response.status,
      });
      return null;
    }

    return (await response.json()) as BackendSubscriptionStatus;
  } catch (error) {
    console.warn("[Polar] Backend subscription status fetch failed", {
      error: error instanceof Error ? error.message : "Unknown error",
    });
    return null;
  }
}

/**
 * Maps the backend status into the gate-state fields, leaving Polar-only
 * snapshot fields (subscription/usage) untouched. When the backend does not
 * report a decisive state, the Polar result is kept unchanged.
 */
export function applyBackendStatus(
  polar: Pick<SubscriptionGateState, "isActive" | "status" | "reason">,
  backend: BackendSubscriptionStatus
): Pick<SubscriptionGateState, "isActive" | "status" | "reason"> {
  if (backend.has_active_subscription) {
    return { isActive: true, status: "active", reason: undefined };
  }

  const reason = backend.reason ?? "";
  if (reason.includes("past_due")) {
    return { isActive: false, status: "past_due", reason: "NO_ACTIVE_SUBSCRIPTION" };
  }
  if (/no active subscription/i.test(reason)) {
    return { isActive: false, status: "none", reason: "NO_ACTIVE_SUBSCRIPTION" };
  }

  return polar;
}

export async function resolveCurrentSubscription(): Promise<SubscriptionGateState> {
  const session = await getMemberSession();
  if (!session?.session_jwt) {
    console.info("[Polar] Subscription state: unauthenticated");
    return {
      isAuthenticated: false,
      isActive: false,
      reason: "UNAUTHENTICATED",
      status: null,
      productId: null,
      meterId: null,
      planId: null,
      subscription: null,
      usage: null,
      backendAvailable: true,
      backendError: null,
    };
  }

  const permissions = await getServerPermissions(session);
  if (!permissions.backendAvailable) {
    console.warn("[Polar] Subscription state: backend unavailable", {
      error: permissions.backendError,
    });

    return {
      isAuthenticated: true,
      isActive: false,
      reason: "BACKEND_UNAVAILABLE",
      status: null,
      productId: null,
      meterId: null,
      planId: null,
      subscription: null,
      usage: null,
      backendAvailable: false,
      backendError: permissions.backendError ?? "Service temporarily unavailable",
    };
  }

  const profile = permissions.profile;
  if (!profile) {
    console.warn("[Polar] Subscription state: profile unavailable");
    return {
      isAuthenticated: true,
      isActive: false,
      reason: "PROFILE_UNAVAILABLE",
      status: null,
      productId: null,
      meterId: null,
      planId: null,
      subscription: null,
      usage: null,
      backendAvailable: true,
      backendError: null,
    };
  }

  if (!permissions.canManageSubscriptions) {
    console.info("[Polar] Subscription state: insufficient permissions", {
      permissions: permissions.permissions,
    });
    return {
      isAuthenticated: true,
      isActive: false,
      reason: "INSUFFICIENT_PERMISSIONS",
      status: null,
      productId: null,
      meterId: null,
      planId: null,
      subscription: null,
      usage: null,
      backendAvailable: true,
      backendError: null,
    };
  }

  const result = await getActiveSubscription({
    externalCustomerId: profile.organization?.organization_id,
    customerEmail: profile.email,
    organizationId: profile.organization?.organization_id,
  });

  // Widen status/reason beyond the Polar result's narrow union so the MP
  // fallback can surface statuses/reasons the Polar SDK never emits.
  let { subscription, isActive, meterId, productId, planId } = result;
  let status: SubscriptionGateState["status"] = result.status;
  let reason: SubscriptionGateState["reason"] = result.reason;

  // MercadoPago fallback: the Polar SDK cannot see MP preapprovals, so when MP
  // is enabled and the Polar path resolves inactive (or the Polar client is
  // unconfigured), consult the provider-agnostic backend status endpoint
  // before declaring the org inactive. Polar-active orgs short-circuit here.
  if (isMercadoPagoEnabled() && (!isActive || reason === "POLAR_UNCONFIGURED")) {
    if (!MERCADOPAGO_CHECKOUT_PLAN_ID) {
      // MP-first deployment without a configured checkout plan id: surface the
      // "billing not configured" state (subscription-tab renders the amber card).
      reason = "MP_UNCONFIGURED";
    } else {
      const backendStatus = await fetchBackendSubscriptionStatus(session.session_jwt);
      if (backendStatus) {
        const mapped = applyBackendStatus({ isActive, status, reason }, backendStatus);
        isActive = mapped.isActive;
        status = mapped.status;
        reason = mapped.reason;
      }
      // Backend unreachable → keep the Polar result (graceful degradation).
    }
  }

  const usage = subscription && isActive ? await getInvoiceUsage(subscription) : null;
  const productName = subscription?.product?.name ?? null;
  const productMetadata =
    subscription && subscription.product?.metadata
      ? (subscription.product.metadata as Record<string, unknown>)
      : null;

  const state: SubscriptionGateState = {
    isAuthenticated: true,
    isActive,
    reason,
    status,
    productId,
    meterId,
    planId: planId ?? null,
      subscription: subscription
        ? {
            id: subscription.id,
            status: subscription.status,
            currentPeriodStart: subscription.currentPeriodStart.toISOString(),
            currentPeriodEnd: subscription.currentPeriodEnd?.toISOString() ?? null,
            cancelAtPeriodEnd: subscription.cancelAtPeriodEnd,
            customerId: subscription.customerId,
            productId: subscription.productId,
            productName,
            productMetadata,
            trialEnd: subscription.trialEnd?.toISOString() ?? null,
            // Additional Polar properties
            trialStart: subscription.trialStart?.toISOString() ?? null,
            recurringInterval: subscription.recurringInterval,
            metadata: subscription.metadata ?? null,
          customFieldData: subscription.customFieldData ?? null,
          customerCancellationReason: subscription.customerCancellationReason ?? null,
          customerCancellationComment: subscription.customerCancellationComment ?? null,
        }
      : null,
    usage: usage
      ? {
          meterId: usage.meterId,
          customerId: usage.customerId,
          included: usage.included,
          used: usage.used,
          remaining: usage.remaining,
          periodStart: usage.periodStart.toISOString(),
          periodEnd: usage.periodEnd.toISOString(),
        }
      : null,
    backendAvailable: true,
    backendError: null,
  };

  console.info("[Polar] Subscription state resolved", {
    isActive: state.isActive,
    reason: state.reason,
    status: state.status,
    productId: state.productId,
    meterId: state.meterId,
    planId: state.planId,
    usage: state.usage
      ? {
          used: state.usage.used,
          remaining: state.usage.remaining,
          included: state.usage.included,
        }
      : undefined,
    backendAvailable: state.backendAvailable,
    backendError: state.backendError,
  });

  return state;
}
