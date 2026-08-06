"use server";

import { getMemberSession } from "@/lib/auth/stytch/server";
import { getServerPermissions } from "@/lib/auth/server-permissions";
import { isMercadoPagoEnabled } from "@/lib/mercadopago/config";
import {
  createActionError,
  createActionSuccess,
  type ActionResult
} from "@/lib/utils/server-action-helpers";

function getGoBackendUrl(): string {
  return process.env.NEXT_PUBLIC_GO_BACKEND_URL ?? process.env.GO_BACKEND_URL ?? "http://localhost:8080";
}

interface CancelMPSubscriptionParams {
  subscriptionId: string;
}

interface CancelResponse {
  status: string;
}

export async function cancelMercadoPagoSubscription(
  params: CancelMPSubscriptionParams
): Promise<ActionResult<CancelResponse>> {
  if (!isMercadoPagoEnabled()) {
    return createActionError(
      "MercadoPago billing is not configured.",
      "Missing MercadoPago access token"
    );
  }

  const session = await getMemberSession();
  if (!session?.session_jwt) {
    return createActionError("Authentication required.");
  }

  const permissions = await getServerPermissions(session);
  if (!permissions.canManageSubscriptions) {
    return createActionError(
      "You do not have access to manage subscriptions.",
      "Missing subscription management permissions"
    );
  }

  try {
    const response = await fetch(`${getGoBackendUrl()}/api/subscriptions/mp-cancel`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${session.session_jwt}`,
      },
      body: JSON.stringify({ subscription_id: params.subscriptionId }),
    });

    if (!response.ok) {
      const errorBody = await response.text();
      return createActionError(
        "Failed to cancel MercadoPago subscription.",
        `Backend returned ${response.status}: ${errorBody}`
      );
    }

    const data: CancelResponse = await response.json();
    return createActionSuccess(data);
  } catch (error) {
    return createActionError(
      "Failed to cancel subscription.",
      error instanceof Error ? error.message : "Unknown error"
    );
  }
}

export async function cancelMPSubscription(
  params: CancelMPSubscriptionParams
): Promise<ActionResult<CancelResponse>> {
  return cancelMercadoPagoSubscription(params);
}
