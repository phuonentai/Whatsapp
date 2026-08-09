"use server";

import { getMemberSession } from "@/lib/auth/stytch/server";
import { isMercadoPagoEnabled } from "@/lib/mercadopago/config";
import {
  createActionError,
  createActionSuccess,
  type ActionResult
} from "@/lib/utils/server-action-helpers";

function getGoBackendUrl(): string {
  return process.env.NEXT_PUBLIC_GO_BACKEND_URL ?? process.env.GO_BACKEND_URL ?? "http://localhost:8080";
}

interface VerifyMPPaymentParams {
  paymentId: string;
}

interface BillingStatusResponse {
  organization_id: number;
  external_id: string;
  has_active_subscription: boolean;
  can_process_invoices: boolean;
  invoice_count: number;
  reason: string;
  checked_at: string;
}

export async function verifyMercadoPagoPayment(
  params: VerifyMPPaymentParams
): Promise<ActionResult<BillingStatusResponse>> {
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

  try {
    const response = await fetch(`${getGoBackendUrl()}/api/subscriptions/verify-mp-payment`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${session.session_jwt}`,
      },
      body: JSON.stringify({ payment_id: params.paymentId }),
    });

    if (!response.ok) {
      const errorBody = await response.text();
      return createActionError(
        "Payment verification failed.",
        `Backend returned ${response.status}: ${errorBody}`
      );
    }

    const data: BillingStatusResponse = await response.json();
    return createActionSuccess(data);
  } catch (error) {
    return createActionError(
      "Failed to verify payment.",
      error instanceof Error ? error.message : "Unknown error"
    );
  }
}
