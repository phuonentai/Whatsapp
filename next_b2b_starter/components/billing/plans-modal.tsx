"use client";

import { useEffect, useMemo, useState, useTransition } from "react";

import type { SubscriptionGateState } from "@/lib/polar/current-subscription";
import { type PolarPlan } from "@/lib/polar/plans";
import { useProductsQuery } from "@/lib/hooks/queries/use-products-query";
import { createCheckout } from "@/lib/actions/billing/create-checkout";
import { createMercadoPagoCheckout } from "@/lib/actions/billing/create-mp-checkout";

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
  const [isPending, startTransition] = useTransition();
  const { data: products, isLoading, error } = useProductsQuery();

  const currentProductId = subscriptionState?.subscription?.productId ?? null;

  useEffect(() => {
    if (!open) {
      setSelectedPlanId(null);
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
    return products.map((plan) => {
      const isCurrent =
        Boolean(subscriptionState?.isActive) &&
        currentProductId === plan.productId;
      return { ...plan, isCurrent };
    });
  }, [currentProductId, subscriptionState?.isActive, products]);

  if (!open) return null;

  const handlePolarCheckout = (plan: PolarPlan) => {
    if (subscriptionState?.isActive) {
      window.alert("Please cancel your current subscription before selecting a new plan.");
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
        setCheckoutError("An unexpected error occurred. Please try again.");
        setSelectedPlanId(null);
        onPlanChangePending?.(false);
      }
    });
  };

  const handleMPCheckout = async (plan: PolarPlan) => {
    if (subscriptionState?.isActive) {
      window.alert("Please cancel your current subscription before selecting a new plan.");
      return;
    }
    onPlanChangePending?.(true);
    setSelectedPlanId(plan.id);
    setCheckoutError(null);

    startTransition(async () => {
      try {
        const result = await createMercadoPagoCheckout({ planId: plan.id });
        if (result.success && result.data.checkoutUrl) {
          window.location.href = result.data.checkoutUrl;
        } else {
          setCheckoutError(result.error || "Failed to create MercadoPago checkout.");
          setSelectedPlanId(null);
          onPlanChangePending?.(false);
        }
      } catch (error) {
        console.error("[PlansModal] MP checkout error:", error);
        setCheckoutError("An unexpected error occurred. Please try again.");
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
          onClick={() => onOpenChange(false)}
          className="absolute right-5 top-5 inline-flex h-9 w-9 items-center justify-center rounded-full border border-gray-200 text-gray-500 transition hover:bg-gray-50 hover:text-gray-700 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-gray-200"
          aria-label="Close plans modal"
        >
          <span className="text-xl leading-none">&times;</span>
        </button>
        <header className="mb-8 space-y-2 pr-12">
          <p className="text-sm font-semibold uppercase tracking-[0.2em] text-gray-500">
            Choose your plan
          </p>
          <h2 className="text-3xl font-semibold text-gray-900">Scale approvals without hitting limits</h2>
          <p className="max-w-2xl text-sm text-gray-600">
            Plans are billed monthly. {mercadopagoEnabled ? "Pay with international card (Polar) or PSE / Nequi / Colombian card (MercadoPago)." : "Payment is processed through Polar."} Seats and invoice quotas update immediately after checkout completes.
          </p>
        </header>

        {isLoading && (
          <div className="flex min-h-[400px] items-center justify-center">
            <div className="text-center">
              <div className="mx-auto mb-4 h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-gray-900" />
              <p className="text-sm text-gray-600">Loading plans...</p>
            </div>
          </div>
        )}

        {error && (
          <div className="flex min-h-[400px] items-center justify-center">
            <div className="text-center">
              <p className="text-sm text-red-600">Failed to load plans. Please try again.</p>
              <button
                type="button"
                onClick={() => window.location.reload()}
                className="mt-4 rounded-full bg-gray-900 px-4 py-2 text-sm font-semibold text-white hover:bg-gray-800"
              >
                Retry
              </button>
            </div>
          </div>
        )}

        {checkoutError && (
          <div className="mb-6 rounded-2xl border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-600">
            <p className="font-semibold">Checkout Error</p>
            <p className="mt-1">{checkoutError}</p>
          </div>
        )}

        {!isLoading && !error && plans.length === 0 && (
          <div className="flex min-h-[400px] items-center justify-center">
            <p className="text-sm text-gray-600">No plans available.</p>
          </div>
        )}

        {!isLoading && !error && plans.length > 0 && (
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
        )}

        {selectedPlanId && (
          <div className="mt-8 flex items-center justify-center gap-3 rounded-2xl border border-gray-200 bg-gray-50 px-5 py-4 text-sm text-gray-600">
            <span className="inline-flex h-3.5 w-3.5 animate-spin rounded-full border-2 border-gray-300 border-t-gray-900" />
            Processing your checkout…
          </div>
        )}
      </div>
    </div>
  );
}

interface PlanCardProps {
  plan: PolarPlan & { isCurrent: boolean };
  disabled: boolean;
  isSelected: boolean;
  isCurrent: boolean;
  onPolarCheckout: () => void;
  onMPCheckout?: () => void;
}

function PlanCard({ plan, disabled, isSelected, isCurrent, onPolarCheckout, onMPCheckout }: PlanCardProps) {
  return (
    <article
      className={`relative flex h-full flex-col justify-between rounded-3xl border p-6 transition ${
        isSelected
          ? "border-gray-900 ring-2 ring-gray-900"
          : isCurrent
            ? "border-emerald-300 ring-1 ring-emerald-200"
            : "border-gray-200 hover:border-gray-300"
      }`}
    >
      {isCurrent ? (
        <span className="absolute right-6 top-6 rounded-full bg-emerald-600 px-3 py-1 text-xs font-semibold uppercase tracking-wide text-white">
          Current Plan
        </span>
      ) : plan.badge && (
        <span className="absolute right-6 top-6 rounded-full bg-gray-900 px-3 py-1 text-xs font-semibold uppercase tracking-wide text-white">
          {plan.badge}
        </span>
      )}

      <div className="space-y-4">
        <div>
          <h3 className="text-xl font-semibold text-gray-900">{plan.name}</h3>
          <p className="mt-1 text-sm text-gray-500">{plan.description}</p>
        </div>

        <div>
          <span className="text-3xl font-semibold text-gray-900">{formatCurrency(plan.price)}</span>
          <span className="ml-1 text-sm text-gray-500">/month</span>
        </div>

        <ul className="space-y-2 text-sm text-gray-600">
          {plan.includedSeats !== null && (
            <li>
              <strong className="font-medium text-gray-900">{plan.includedSeats}</strong> seats included
            </li>
          )}
          {plan.includedInvoices !== null && (
            <li>
              <strong className="font-medium text-gray-900">{plan.includedInvoices.toLocaleString()}</strong> invoices per month
            </li>
          )}
          {plan.benefits.map((benefit) => (
            <li key={benefit}>{benefit}</li>
          ))}
        </ul>
      </div>

      {isCurrent ? (
        <button
          type="button"
          disabled
          className="mt-6 inline-flex w-full items-center justify-center rounded-full px-5 py-2 text-sm font-semibold cursor-not-allowed border border-emerald-100 bg-emerald-50 text-emerald-600"
        >
          Current plan
        </button>
      ) : (
        <div className="mt-6 flex flex-col gap-2">
          <button
            type="button"
            onClick={onPolarCheckout}
            disabled={disabled}
            className="inline-flex w-full items-center justify-center rounded-full bg-gray-900 px-5 py-2 text-sm font-semibold text-white shadow hover:bg-gray-800 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-gray-900 disabled:cursor-not-allowed disabled:bg-gray-200 disabled:text-gray-500"
          >
            {isSelected ? "Processing…" : "International card"}
          </button>
          {onMPCheckout && (
            <button
              type="button"
              onClick={onMPCheckout}
              disabled={disabled}
              className="inline-flex w-full items-center justify-center rounded-full border border-blue-600 bg-white px-5 py-2 text-sm font-semibold text-blue-600 shadow hover:bg-blue-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600 disabled:cursor-not-allowed disabled:border-gray-200 disabled:text-gray-500 disabled:bg-gray-50"
            >
              {isSelected ? "Processing…" : "PSE / Nequi / Colombian card"}
            </button>
          )}
        </div>
      )}
    </article>
  );
}

function formatCurrency(amount: number) {
  return new Intl.NumberFormat(undefined, {
    style: "currency",
    currency: "USD",
  }).format(amount);
}
