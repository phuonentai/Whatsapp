import { getMemberSession } from "@/lib/auth/stytch/server";
import { getServerPermissions } from "@/lib/auth/server-permissions";
import { getActiveSubscription } from "@/lib/polar/subscription";
import { getInvoiceUsage } from "@/lib/polar/usage";

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

  const { subscription, isActive, status, meterId, productId, planId, reason } = result;

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
