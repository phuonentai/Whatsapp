"use client";

import type { PolarPlan } from "@/lib/polar/plans";
import { ui } from "@/lib/copy/ui";

interface PlanCardProps {
  plan: PolarPlan & { isCurrent: boolean };
  disabled: boolean;
  isSelected: boolean;
  isCurrent: boolean;
  onPolarCheckout: () => void;
  onMPCheckout?: () => void;
}

export function PlanCard({
  plan,
  disabled,
  isSelected,
  isCurrent,
  onPolarCheckout,
  onMPCheckout,
}: PlanCardProps) {
  const aiCredits = getAiCredits(plan);

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
          {ui.billing.currentPlan}
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
          <span className="ml-1 text-sm text-gray-500">
            {plan.interval === "year" ? ui.billing.perYear : ui.billing.perMonth}
          </span>
        </div>

        <ul className="space-y-2 text-sm text-gray-600">
          {plan.includedSeats !== null && (
            <li>
              <strong className="font-medium text-gray-900">{plan.includedSeats}</strong> {ui.billing.seatsIncluded}
            </li>
          )}
          {plan.includedInvoices !== null && (
            <li>
              <strong className="font-medium text-gray-900">{plan.includedInvoices.toLocaleString()}</strong> {ui.billing.invoicesPerMonth}
            </li>
          )}
          {aiCredits !== null && (
            <li>
              <strong className="font-medium text-gray-900">{aiCredits.toLocaleString()}</strong> {ui.billing.aiCreditsPerPeriod}
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
          {ui.billing.currentPlan}
        </button>
      ) : (
        <div className="mt-6 flex flex-col gap-2">
          <button
            type="button"
            onClick={onPolarCheckout}
            disabled={disabled}
            className="inline-flex w-full items-center justify-center rounded-full bg-gray-900 px-5 py-2 text-sm font-semibold text-white shadow hover:bg-gray-800 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-gray-900 disabled:cursor-not-allowed disabled:bg-gray-200 disabled:text-gray-500"
          >
            {isSelected ? ui.billing.processingDots : ui.billing.internationalCard}
          </button>
          {onMPCheckout && (
            <button
              type="button"
              onClick={onMPCheckout}
              disabled={disabled}
              className="inline-flex w-full items-center justify-center rounded-full border border-blue-600 bg-white px-5 py-2 text-sm font-semibold text-blue-600 shadow hover:bg-blue-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600 disabled:cursor-not-allowed disabled:border-gray-200 disabled:text-gray-500 disabled:bg-gray-50"
            >
              {isSelected ? ui.billing.processingDots : ui.billing.mpCard}
            </button>
          )}
        </div>
      )}
    </article>
  );
}

export function getAiCredits(plan: PolarPlan): number | null {
  const value = plan.metadata?.ai_credits_max;
  return typeof value === "number" ? value : null;
}

function formatCurrency(amount: number) {
  return new Intl.NumberFormat(undefined, {
    style: "currency",
    currency: "USD",
  }).format(amount);
}
