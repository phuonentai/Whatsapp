import { redirect } from "next/navigation";
import { verifyPayment } from "@/lib/actions/billing/verify-payment";
import { verifyMercadoPagoPayment } from "@/lib/actions/billing/verify-mp-payment";

interface DashboardPageProps {
  searchParams: Promise<{ checkout_id?: string; payment_id?: string; preference_id?: string }>;
}

export default async function DashboardPage({ searchParams }: DashboardPageProps) {
  const params = await searchParams;
  const checkoutId = params.checkout_id;
  const paymentId = params.payment_id ?? params.preference_id;

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

  redirect("/dashboard/settings");
}
