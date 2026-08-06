import type { Polar } from "@polar-sh/sdk";
import type { Customer } from "@polar-sh/sdk/models/components/customer";
import type { Subscription } from "@polar-sh/sdk/models/components/subscription";
import type { SubscriptionsListRequest } from "@polar-sh/sdk/models/operations/subscriptionslist";

import { getPolarClient } from "@/lib/polar/client";
import { POLAR_METER_ID } from "@/lib/polar/config";

const ACTIVE_STATUSES = new Set<Subscription["status"]>(["active", "trialing"]);

export interface SubscriptionLookupInput {
  externalCustomerId?: string | null;
  customerEmail?: string | null;
  organizationId?: string | null;
}

export interface SubscriptionStatusResult {
  subscription: Subscription | null;
  customer: Customer | null;
  isActive: boolean;
  status: Subscription["status"] | null;
  meterId: string | null;
  productId: string | null;
  planId: string | null;
  reason?: "POLAR_UNCONFIGURED" | "CUSTOMER_NOT_FOUND" | "NO_ACTIVE_SUBSCRIPTION" | "UNKNOWN_ERROR";
}

export async function getActiveSubscription(
  input: SubscriptionLookupInput
): Promise<SubscriptionStatusResult> {
  const client = getPolarClient();
  if (!client) {
    console.warn("[Polar] Subscription lookup skipped - Polar client unavailable");
    return {
      subscription: null,
      customer: null,
      isActive: true,
      status: null,
      meterId: POLAR_METER_ID ?? null,
      productId: null,
      planId: null,
      reason: "POLAR_UNCONFIGURED",
    };
  }

  const externalCustomerId =
    input.externalCustomerId ?? input.organizationId ?? null;

  const request: SubscriptionsListRequest = {
    active: true,
    limit: 1,
  };

  if (externalCustomerId) {
    request.externalCustomerId = externalCustomerId;
  }

  let customer: Customer | null = null;

  try {
    if (!request.externalCustomerId && input.customerEmail) {
      customer = await findCustomerByEmail(client, input.customerEmail);
      if (customer) {
        request.customerId = customer.id;
        console.debug("[Polar] Found customer by email", {
          customerId: customer.id,
          email: input.customerEmail,
        });
      }
    }

    const iterator = await client.subscriptions.list(request);
    const firstPage = iterator.result;
    const subscription = firstPage.items[0] ?? null;

    if (!customer && subscription) {
      customer = subscription.customer;
    }

    if (!subscription) {
      console.info("[Polar] No active subscription found", {
        hasCustomer: Boolean(customer),
        externalCustomerId,
      });
      return {
        subscription: null,
        customer,
        isActive: false,
        status: null,
        meterId: POLAR_METER_ID ?? null,
        productId: null,
        planId: null,
        reason: customer ? "NO_ACTIVE_SUBSCRIPTION" : "CUSTOMER_NOT_FOUND",
      };
    }

    const isActive = ACTIVE_STATUSES.has(subscription.status);
    const metadata = (subscription.metadata ?? {}) as Record<string, unknown>;
    const rawPlanId = metadata["plan_id"];
    const planId =
      typeof rawPlanId === "string" && rawPlanId.trim().length > 0 ? rawPlanId.trim() : null;

    // Log complete metadata for debugging
    console.info("[Polar] Subscription lookup result", {
      subscriptionId: subscription.id,
      status: subscription.status,
      isActive,
      customerId: subscription.customerId,
      planId,
      productId: subscription.productId,
      // Full metadata dump
      subscriptionMetadata: subscription.metadata,
      customerMetadata: subscription.customer?.metadata,
      productName: subscription.product?.name,
      productMetadata: subscription.product?.metadata,
      // Additional important fields
      trialStart: subscription.trialStart,
      trialEnd: subscription.trialEnd,
      cancelAtPeriodEnd: subscription.cancelAtPeriodEnd,
      currentPeriodStart: subscription.currentPeriodStart,
      currentPeriodEnd: subscription.currentPeriodEnd,
      recurringInterval: subscription.recurringInterval,
      // NOTE: subscription.meters[] is always empty - use client.meters.customers() API instead
    });

    return {
      subscription,
      customer,
      isActive,
      status: subscription.status,
      meterId: POLAR_METER_ID ?? null,
      productId: subscription.productId ?? null,
      planId,
      reason: isActive ? undefined : "NO_ACTIVE_SUBSCRIPTION",
    };
  } catch (error) {
    console.error("[Polar] Failed to load subscription", error);
    return {
      subscription: null,
      customer,
      isActive: false,
      status: null,
      meterId: POLAR_METER_ID ?? null,
      productId: null,
      planId: null,
      reason: "UNKNOWN_ERROR",
    };
  }
}

async function findCustomerByEmail(
  client: Polar,
  email: string
): Promise<Customer | null> {
  try {
    console.debug("[Polar] Searching for customer by email", { email });
    const iterator = await client.customers.list({
      email,
      limit: 1,
    });
    const result = iterator.result;
    if (result.items[0]) {
      console.debug("[Polar] Customer lookup succeeded", {
        customerId: result.items[0].id,
      });
    } else {
      console.info("[Polar] Customer lookup returned empty", { email });
    }
    return result.items[0] ?? null;
  } catch (error) {
    console.error("[Polar] Failed to find customer by email", error);
    return null;
  }
}
