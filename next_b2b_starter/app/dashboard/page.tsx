import { redirect } from "next/navigation";
import { verifyPayment } from "@/lib/actions/billing/verify-payment";
import { verifyMercadoPagoPayment } from "@/lib/actions/billing/verify-mp-payment";
import { DashboardHome } from "./components/dashboard-home";

interface DashboardPageProps {
  searchParams: Promise<{
    checkout_id?: string;
    payment_id?: string;
    preference_id?: string;
    preapproval_id?: string;
  }>;
}

export default async function DashboardPage({ searchParams }: DashboardPageProps) {
  const params = await searchParams;
  const checkoutId = params.checkout_id;
  const paymentId = params.payment_id ?? params.preference_id;
  const preapprovalId = params.preapproval_id;

  if (checkoutId) {
    const result = await verifyPayment(checkoutId);

    if (result.success) {
      console.info("[Dashboard] Polar payment verified successfully", {
        sessionId: checkoutId,
        hasActiveSubscription: result.data.has_active_subscription,
      });
      redirect("/dashboard/settings?view=subscription&payment_verified=true");
    } else {
      console.error("[Dashboard] Polar payment verification failed", {
        sessionId: checkoutId,
        error: result.error,
      });
      redirect(`/dashboard/settings?view=subscription&payment_error=true`);
    }
  }

  // MercadoPago preapproval-only return (no payment id yet): the subscription
  // settles via the subscription_authorized webhook or a later payment, so
  // land on the subscription view without a false error banner.
  if (preapprovalId && !paymentId) {
    console.info("[Dashboard] MercadoPago preapproval return without payment", {
      preapprovalId,
    });
    redirect("/dashboard/settings?view=subscription");
  }

  if (paymentId) {
    const result = await verifyMercadoPagoPayment({ paymentId });

    if (result.success) {
      console.info("[Dashboard] MercadoPago payment verified successfully", {
        paymentId,
        hasActiveSubscription: result.data.has_active_subscription,
      });
      redirect("/dashboard/settings?view=subscription&payment_verified=true");
    } else {
      console.error("[Dashboard] MercadoPago payment verification failed", {
        paymentId,
        error: result.error,
      });
      redirect(`/dashboard/settings?view=subscription&payment_error=true`);
    }
  }

  return <DashboardHome />;
}
