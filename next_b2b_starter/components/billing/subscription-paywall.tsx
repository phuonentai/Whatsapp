"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";

import type { SubscriptionGateState } from "@/lib/polar/current-subscription";
import { PlansModal } from "@/components/billing/plans-modal";
import { getPlanById, getPlanByProductId, type PolarPlan } from "@/lib/polar/plans";
import { useSubscriptionQuery } from "@/lib/hooks/queries/use-subscription-query";
import { useProductsQuery } from "@/lib/hooks/queries/use-products-query";
import { Skeleton } from "@/components/ui/skeleton";
import { usePermissions } from "@/lib/hooks/use-permissions";
import { PERMISSIONS } from "@/lib/auth/permissions";
import { isMercadoPagoEnabled } from "@/lib/mercadopago/config";
import { ui } from "@/lib/copy/ui";

interface SubscriptionPaywallProps {
  // No props required - component fetches its own data
}

export function SubscriptionPaywall({}: SubscriptionPaywallProps = {}) {
  const router = useRouter();
  const {
    hasPermission,
    isInitialized: permissionsReady,
  } = usePermissions();
  const hasSubscriptionPermission = hasPermission(PERMISSIONS.ORG_MANAGE);
  const shouldLoadSubscription = permissionsReady && hasSubscriptionPermission;
  const mercadopagoEnabled = isMercadoPagoEnabled();

  const [isPlansOpen, setPlansOpen] = useState(false);

  const {
    data: state,
    isLoading,
    error: subscriptionError,
  } = useSubscriptionQuery({
    enabled: shouldLoadSubscription,
  });
  const { data: products, error: productsError } = useProductsQuery({
    enabled: shouldLoadSubscription,
  });

  // Redirect to dashboard if subscription becomes active
  useEffect(() => {
    if (state?.isActive) {
      router.replace("/dashboard");
    }
  }, [state?.isActive, router]);

  if (permissionsReady && !hasSubscriptionPermission) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-gray-50 px-6 py-12">
        <div className="w-full max-w-lg space-y-4 text-center">
          <h1 className="text-2xl font-semibold text-gray-900">
            {ui.billing.restrictedTitle}
          </h1>
          <p className="text-sm text-gray-600">
            {ui.billing.restrictedBody}
          </p>
        </div>
      </main>
    );
  }

  const loadError = subscriptionError ?? productsError;

  if (shouldLoadSubscription && loadError) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-gray-50 px-6 py-12">
        <div className="w-full max-w-lg space-y-4 text-center">
          <h1 className="text-2xl font-semibold text-gray-900">
            {ui.billing.loadSubscriptionTitle}
          </h1>
          <p className="text-sm text-gray-600">
            {loadError.message || ui.billing.refreshOrSupport}
          </p>
        </div>
      </main>
    );
  }

  if (!shouldLoadSubscription || isLoading || !state) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-gray-50 px-6 py-12">
        <div className="w-full max-w-4xl space-y-6">
          <Skeleton className="h-96 w-full rounded-3xl" />
        </div>
      </main>
    );
  }

  // If subscription is active but haven't redirected yet, show loading
  if (state.isActive) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-gray-50 px-6 py-12">
        <div className="flex flex-col items-center gap-3">
          <div className="h-10 w-10 animate-spin rounded-full border-4 border-gray-200 border-t-gray-900" />
          <p className="text-sm text-gray-600">{ui.billing.redirecting}</p>
        </div>
      </main>
    );
  }

  const usage = state.usage;

  const included = usage?.included ?? 1000; // Default fallback
  const used = usage?.used ?? 0;
  const remaining = usage?.remaining ?? included;
  const plan = resolvePlanFromState(state, products ?? []);
  const planPrice = plan?.price ?? null;
  const formattedPrice = planPrice != null ? formatUsd(planPrice) : null;
  const interval = plan?.interval === "month" ? ui.billing.perMonth : "per billing period";

  const contactHref =
    process.env.NEXT_PUBLIC_CONTACT_EMAIL ||
    process.env.NOTIFICATION_EMAIL ||
    "mailto:info@yourdomain.com";

  console.info("[Polar] Rendering SubscriptionPaywall", {
    isActive: state.isActive,
    reason: state.reason,
    status: state.status,
    included,
    used,
    remaining,
    plan: plan?.id ?? "unknown",
    planPrice,
  });

  const featureList = buildFeatureList({
    includedInvoices: included,
    plan,
  });

  return (
    <main className="flex min-h-screen items-center justify-center bg-gray-50 px-6 py-12">
      <div className="w-full max-w-4xl">
        <div className="grid gap-6 lg:grid-cols-[2fr,1fr]">
          <section className="flex flex-col justify-between rounded-3xl bg-white p-10 shadow-lg ring-1 ring-gray-200">
            <div>
              <span className="inline-flex items-center rounded-full bg-gray-900 px-3 py-1 text-xs font-semibold uppercase tracking-wide text-white">
                {plan?.name ?? ui.billing.proPlan}
              </span>
              <h1 className="mt-6 text-3xl font-semibold text-gray-900 sm:text-4xl">
                {ui.billing.unlockTitle}
              </h1>
              <p className="mt-4 text-base text-gray-600">
                {ui.billing.unlockBody}
              </p>

              <div className="mt-8 grid gap-4 sm:grid-cols-2">
                {featureList.map((feature) => (
                  <Feature
                    key={feature.title}
                    title={feature.title}
                    description={feature.description}
                  />
                ))}
              </div>
            </div>

            <div className="mt-10 flex flex-col gap-3 sm:flex-row sm:items-center">
              <button
                type="button"
                onClick={() => setPlansOpen(true)}
                className="inline-flex items-center justify-center rounded-md bg-gray-900 px-6 py-3 text-sm font-semibold text-white shadow-sm transition hover:bg-gray-800 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-gray-900"
              >
                {ui.billing.viewPlans}
              </button>
              <a
                href={contactHref}
                className="inline-flex items-center justify-center rounded-md border border-gray-200 px-6 py-3 text-sm font-semibold text-gray-700 transition hover:bg-gray-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-gray-200"
              >
                {ui.billing.talkToSales}
              </a>
            </div>
          </section>

          <aside className="rounded-3xl bg-gray-900 p-8 text-white shadow-lg ring-1 ring-gray-900/10">
            <p className="text-sm uppercase tracking-[0.16em] text-gray-400">
              {ui.billing.planSnapshot}
            </p>
            <p className="mt-4 text-4xl font-semibold">
              {formattedPrice ?? ui.billing.contactSales}
            </p>
            <p className="text-sm text-gray-400">
              {formattedPrice ? `${interval} · ${ui.billing.cancelAnytime}` : ui.billing.activateHelp}
            </p>

            <dl className="mt-8 space-y-4">
              <UsageItem label={ui.billing.invoicesRemaining} value={remaining} total={included} />
              <UsageItem label={ui.billing.invoicesUsedPeriod} value={used} total={included} />
              {state.subscription?.trialEnd && (
                <div className="rounded-lg border border-white/10 bg-white/5 p-4">
                  <dt className="text-sm text-gray-300">{ui.billing.trialEnds}</dt>
                  <dd className="mt-1 text-lg font-semibold text-white">
                    {new Date(state.subscription.trialEnd).toLocaleDateString()}
                  </dd>
                </div>
              )}
            </dl>

            {state.reason === "NO_ACTIVE_SUBSCRIPTION" && (
              <p className="mt-8 rounded-lg border border-amber-500/20 bg-amber-500/10 px-4 py-3 text-sm text-amber-100">
                {mercadopagoEnabled
                  ? ui.billing.noActiveSubMp
                  : ui.billing.noActiveSubPolar}
              </p>
            )}

            {state.reason === "CUSTOMER_NOT_FOUND" && (
              <p className="mt-8 rounded-lg border border-blue-500/20 bg-blue-500/10 px-4 py-3 text-sm text-blue-100">
                {mercadopagoEnabled
                  ? ui.billing.customerNotFoundMp
                  : ui.billing.customerNotFoundPolar}
              </p>
            )}

            {state.reason === "UNKNOWN_ERROR" && (
              <p className="mt-8 rounded-lg border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-100">
                {ui.billing.unknownErrorBody}
              </p>
            )}
          </aside>
        </div>
      </div>
      <PlansModal
        open={isPlansOpen}
        onOpenChange={setPlansOpen}
        subscriptionState={state}
        mercadopagoEnabled={mercadopagoEnabled}
      />
    </main>
  );
}

interface FeatureProps {
  title: string;
  description: string;
}

function Feature({ title, description }: FeatureProps) {
  return (
    <div className="rounded-2xl border border-gray-200 p-4">
      <h3 className="text-sm font-semibold text-gray-900">{title}</h3>
      <p className="mt-1 text-sm text-gray-500">{description}</p>
    </div>
  );
}

interface UsageItemProps {
  label: string;
  value: number;
  total: number;
}

function UsageItem({ label, value, total }: UsageItemProps) {
  const percentage = Math.min(Math.round((value / Math.max(total, 1)) * 100), 100);
  return (
    <div>
      <dt className="text-sm text-gray-400">{label}</dt>
      <dd className="mt-1 flex items-center justify-between text-lg font-semibold text-white">
        <span>{value.toLocaleString()}</span>
        <span className="text-sm font-medium text-gray-400">{ui.billing.of} {total.toLocaleString()}</span>
      </dd>
      <div className="mt-2 h-2 rounded-full bg-gray-800">
        <div
          className="h-full rounded-full bg-gradient-to-r from-violet-400 to-indigo-400"
          style={{ width: `${percentage}%` }}
        />
      </div>
    </div>
  );
}

function resolvePlanFromState(state: SubscriptionGateState, products: PolarPlan[]): PolarPlan | null {
  if (state.planId) {
    const byId = getPlanById(products, state.planId);
    if (byId) {
      return byId;
    }
  }

  if (state.subscription?.productId) {
    const bySubscriptionProduct = getPlanByProductId(products, state.subscription.productId);
    if (bySubscriptionProduct) {
      return bySubscriptionProduct;
    }
  }

  if (state.productId) {
    const byProduct = getPlanByProductId(products, state.productId);
    if (byProduct) {
      return byProduct;
    }
  }

  return null;
}

function buildFeatureList({
  includedInvoices,
  plan,
}: {
  includedInvoices: number;
  plan: PolarPlan | null;
}) {
  const features: Array<{ title: string; description: string }> = [
    {
      title: `${includedInvoices.toLocaleString()} ${ui.billing.invoicesMonthIncluded}`,
      description: ui.billing.invoicesMonthIncludedDesc,
    },
  ];

  if (plan?.price != null) {
    features.push({
      title: `${formatUsd(plan.price)} ${ui.billing.flatSubscription}`,
      description: ui.billing.flatSubscriptionDesc,
    });
  }

  if (plan?.benefits?.length) {
    for (const benefit of plan.benefits) {
      features.push({
        title: benefit,
        description: ui.billing.includedWithPlan,
      });
    }
  } else {
    features.push(
      {
        title: ui.billing.approvalsAnomaly,
        description: ui.billing.approvalsAnomalyDesc,
      },
      {
        title: ui.billing.exportReadyPayments,
        description: ui.billing.exportReadyPaymentsDesc,
      }
    );
  }

  return features;
}

function formatUsd(amount: number) {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    maximumFractionDigits: amount % 1 === 0 ? 0 : 2,
  }).format(amount);
}
