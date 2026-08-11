"use client";

import { useEffect, useMemo, useState, useTransition } from "react";

import type { SubscriptionGateState } from "@/lib/polar/current-subscription";
import { type PolarPlan } from "@/lib/polar/plans";
import { useProductsQuery } from "@/lib/hooks/queries/use-products-query";
import { createCheckout } from "@/lib/actions/billing/create-checkout";
import { createMercadoPagoCheckout } from "@/lib/actions/billing/create-mp-checkout";
import { ConfirmDialog } from "@/components/crm/confirm-dialog";
import { ui } from "@/lib/copy/ui";
import { PlanCard } from "./plan-card";
import { PlansComparison } from "./plans-comparison";

interface PlansModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  subscriptionState?: SubscriptionGateState | null;
  onPlanChangePending?: (pending: boolean) => void;
  mercadopagoEnabled?: boolean;
}

export function PlansModal({
  open,
  onOpenChange,
  subscriptionState,
  onPlanChangePending,
  mercadopagoEnabled = false,
}: PlansModalProps) {
  const [selectedPlanId, setSelectedPlanId] = useState<string | null>(null);
  const [checkoutError, setCheckoutError] = useState<string | null>(null);
  const [blockingDialogOpen, setBlockingDialogOpen] = useState(false);
  const [isPending, startTransition] = useTransition();
  const [selectedInterval, setSelectedInterval] = useState<"month" | "year">("month");
  const { data: products, isLoading, error } = useProductsQuery();

  const currentProductId = subscriptionState?.subscription?.productId ?? null;

  const intervalOptions = useMemo(() => {
    if (!products) return [] as Array<"month" | "year">;
    const intervals = new Set(products.map((plan) => plan.interval));
    return (["month", "year"] as const).filter((interval) => intervals.has(interval));
  }, [products]);

  useEffect(() => {
    if (intervalOptions.length > 0 && !intervalOptions.includes(selectedInterval)) {
      setSelectedInterval(intervalOptions[0]);
    }
  }, [intervalOptions, selectedInterval]);

  const handleClose = () => {
    setSelectedPlanId(null);
    setCheckoutError(null);
    setBlockingDialogOpen(false);
    onOpenChange(false);
  };

  useEffect(() => {
    if (!open) {
      document.body.style.removeProperty("overflow");
      return;
    }
    document.body.style.setProperty("overflow", "hidden");
  }, [open]);

  useEffect(() => {
    return () => {
      onPlanChangePending?.(false);
    };
  }, [onPlanChangePending]);

  useEffect(() => {
    return () => {
      document.body.style.removeProperty("overflow");
    };
  }, []);

  const plans = useMemo(() => {
    if (!products) return [];
    return products
      .filter((plan) => plan.interval === selectedInterval)
      .map((plan) => {
        const isCurrent =
          Boolean(subscriptionState?.isActive) &&
          currentProductId === plan.productId;
        return { ...plan, isCurrent };
      });
  }, [currentProductId, selectedInterval, subscriptionState?.isActive, products]);

  if (!open) return null;

  const handlePolarCheckout = (plan: PolarPlan) => {
    if (subscriptionState?.isActive) {
      setBlockingDialogOpen(true);
      return;
    }
    onPlanChangePending?.(true);
    setSelectedPlanId(plan.id);
    setCheckoutError(null);

    startTransition(async () => {
      try {
        const result = await createCheckout({ planId: plan.id });
        if (!result.success) {
          setCheckoutError(result.error);
          setSelectedPlanId(null);
          onPlanChangePending?.(false);
        }
      } catch (error) {
        if (error instanceof Error && error.message === "NEXT_REDIRECT") return;
        console.error("[PlansModal] Polar checkout error:", error);
        setCheckoutError(ui.common.unexpectedError);
        setSelectedPlanId(null);
        onPlanChangePending?.(false);
      }
    });
  };

  const handleMPCheckout = async (plan: PolarPlan) => {
    if (subscriptionState?.isActive) {
      setBlockingDialogOpen(true);
      return;
    }
    onPlanChangePending?.(true);
    setSelectedPlanId(plan.id);
    setCheckoutError(null);

    startTransition(async () => {
      try {
        const result = await createMercadoPagoCheckout({ planId: plan.id });
        if (!result.success) {
          setCheckoutError(result.error || ui.billing.checkoutCreationFailed);
          setSelectedPlanId(null);
          onPlanChangePending?.(false);
        } else if (result.data.checkoutUrl) {
          window.location.href = result.data.checkoutUrl;
        } else {
          setCheckoutError(ui.billing.checkoutCreationFailed);
          setSelectedPlanId(null);
          onPlanChangePending?.(false);
        }
      } catch (error) {
        console.error("[PlansModal] MP checkout error:", error);
        setCheckoutError(ui.common.unexpectedError);
        setSelectedPlanId(null);
        onPlanChangePending?.(false);
      }
    });
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 px-4 py-10 backdrop-blur-sm">
      <div className="relative w-full max-w-4xl rounded-3xl bg-white p-8 shadow-2xl ring-1 ring-gray-200">
        <button
          type="button"
          onClick={handleClose}
          className="absolute right-5 top-5 inline-flex h-9 w-9 items-center justify-center rounded-full border border-gray-200 text-gray-500 transition hover:bg-gray-50 hover:text-gray-700 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-gray-200"
          aria-label={ui.billing.closePlansAria}
        >
          <span className="text-xl leading-none">&times;</span>
        </button>
        <header className="mb-8 space-y-2 pr-12">
          <p className="text-sm font-semibold uppercase tracking-[0.2em] text-gray-500">
            {ui.billing.plansEyebrow}
          </p>
          <h2 className="text-3xl font-semibold text-gray-900">{ui.billing.plansTitle}</h2>
          <p className="max-w-2xl text-sm text-gray-600">
            {ui.billing.monthlyBilling}{" "}
            {mercadopagoEnabled ? ui.billing.mpPaymentNote : ui.billing.polarPaymentNote}{" "}
            {ui.billing.quotasUpdateNote}
          </p>
        </header>

        {isLoading && (
          <div className="flex min-h-[400px] items-center justify-center">
            <div className="text-center">
              <div className="mx-auto mb-4 h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-gray-900" />
              <p className="text-sm text-gray-600">{ui.billing.loadingPlans}</p>
            </div>
          </div>
        )}

        {error && (
          <div className="flex min-h-[400px] items-center justify-center">
            <div className="text-center">
              <p className="text-sm text-red-600">{ui.billing.loadPlansError}</p>
              <button
                type="button"
                onClick={() => window.location.reload()}
                className="mt-4 rounded-full bg-gray-900 px-4 py-2 text-sm font-semibold text-white hover:bg-gray-800"
              >
                {ui.common.retry}
              </button>
            </div>
          </div>
        )}

        {subscriptionState?.isActive && (
          <div className="mb-6 rounded-2xl border border-amber-200 bg-amber-50 px-5 py-4 text-sm text-amber-800">
            <p className="font-semibold">{ui.billing.activeSubscription}</p>
            <p className="mt-1">
              {ui.billing.activeSubscriptionHint}
            </p>
          </div>
        )}

        {intervalOptions.length > 1 && (
          <div
            className="mb-6 inline-flex rounded-full border border-gray-200 bg-gray-50 p-1"
            role="group"
            aria-label={ui.billing.intervalToggleAria}
          >
            {intervalOptions.map((interval) => (
              <button
                key={interval}
                type="button"
                onClick={() => setSelectedInterval(interval)}
                className={`rounded-full px-4 py-1.5 text-sm font-semibold transition ${
                  selectedInterval === interval
                    ? "bg-gray-900 text-white shadow"
                    : "text-gray-600 hover:text-gray-900"
                }`}
              >
                {interval === "month" ? ui.billing.intervalMonthly : ui.billing.intervalAnnual}
              </button>
            ))}
          </div>
        )}

        {checkoutError && (
          <div className="mb-6 rounded-2xl border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-600">
            <p className="font-semibold">{ui.billing.checkoutError}</p>
            <p className="mt-1">{checkoutError}</p>
          </div>
        )}

        {!isLoading && !error && plans.length === 0 && (
          <div className="flex min-h-[400px] items-center justify-center">
            <p className="text-sm text-gray-600">{ui.billing.noPlans}</p>
          </div>
        )}

        {!isLoading && !error && plans.length > 0 && (
          <>
            <div className="grid gap-4 lg:grid-cols-2">
              {plans.map((plan) => (
                <PlanCard
                  key={plan.id}
                  plan={plan}
                  disabled={Boolean(selectedPlanId) || isPending}
                  isSelected={selectedPlanId === plan.id}
                  isCurrent={plan.isCurrent}
                  onPolarCheckout={() => handlePolarCheckout(plan)}
                  onMPCheckout={mercadopagoEnabled ? () => handleMPCheckout(plan) : undefined}
                />
              ))}
            </div>
            {plans.length > 1 && <PlansComparison plans={plans} />}
          </>
        )}

        {selectedPlanId && (
          <div className="mt-8 flex items-center justify-center gap-3 rounded-2xl border border-gray-200 bg-gray-50 px-5 py-4 text-sm text-gray-600">
            <span className="inline-flex h-3.5 w-3.5 animate-spin rounded-full border-2 border-gray-300 border-t-gray-900" />
            {ui.billing.processingCheckout}
          </div>
        )}
      </div>

      <ConfirmDialog
        open={blockingDialogOpen}
        onOpenChange={setBlockingDialogOpen}
        title={ui.billing.activeSubscription}
        description={ui.billing.activeSubscriptionHint}
        confirmLabel={ui.billing.understood}
        cancelLabel={ui.billing.close}
        destructive={false}
        onConfirm={() => setBlockingDialogOpen(false)}
      />
    </div>
  );
}
