"use server";

import { redirect } from "next/navigation";
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

interface CreateMPCheckoutParams {
  planId?: string;
}

interface MPCheckoutData {
  checkoutUrl?: string;
  message: string;
  checkedAt: string;
}

export async function createMercadoPagoCheckout(
  params?: CreateMPCheckoutParams
): Promise<ActionResult<MPCheckoutData> | never> {
  if (!isMercadoPagoEnabled()) {
    return createActionError(
      "MercadoPago billing is not configured.",
      "Missing NEXT_PUBLIC_MERCADOPAGO_PLAN_ID configuration"
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
    const planId = params?.planId ?? process.env.NEXT_PUBLIC_MERCADOPAGO_CHECKOUT_PLAN_ID ?? "default";

    const response = await fetch(`${getGoBackendUrl()}/api/subscriptions/create-mp-checkout`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${session.session_jwt}`,
      },
      body: JSON.stringify({ plan_id: planId }),
    });

    if (!response.ok) {
      const errorBody = await response.text();
      return createActionError(
        "Failed to create MercadoPago checkout.",
        `Backend returned ${response.status}: ${errorBody}`
      );
    }

    const data = await response.json();
    return createActionSuccess(data);
  } catch (error) {
    return createActionError(
      "Failed to create MercadoPago checkout.",
      error instanceof Error ? error.message : "Unknown error"
    );
  }
}
