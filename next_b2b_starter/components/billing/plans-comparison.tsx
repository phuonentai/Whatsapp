"use client";

import type { PolarPlan } from "@/lib/polar/plans";
import { ui } from "@/lib/copy/ui";
import { getAiCredits } from "./plan-card";

interface PlansComparisonProps {
  plans: Array<PolarPlan & { isCurrent: boolean }>;
}

interface ComparisonRow {
  label: string;
  value: (plan: PolarPlan) => number | null;
}

const ROWS: ComparisonRow[] = [
  { label: ui.billing.comparisonSeats, value: (plan) => plan.includedSeats },
  { label: ui.billing.comparisonInvoices, value: (plan) => plan.includedInvoices },
  { label: ui.billing.comparisonAiCredits, value: getAiCredits },
];

export function PlansComparison({ plans }: PlansComparisonProps) {
  return (
    <div className="mt-10 overflow-x-auto">
      <h3 className="text-sm font-semibold uppercase tracking-[0.2em] text-gray-500">
        {ui.billing.comparisonTitle}
      </h3>
      <table className="mt-4 w-full min-w-[560px] border-collapse text-sm">
        <thead>
          <tr className="border-b border-gray-200">
            <th className="py-3 pr-4 text-left align-bottom text-xs font-semibold uppercase tracking-[0.2em] text-gray-500">
              {ui.billing.plansEyebrow}
            </th>
            {plans.map((plan) => (
              <th key={plan.id} className="px-4 py-3 text-left align-bottom" scope="col">
                <p className="font-semibold text-gray-900">{plan.name}</p>
                <p className="mt-0.5 text-xs font-normal text-gray-500">
                  {formatCurrency(plan.price)}{" "}
                  {plan.interval === "year" ? ui.billing.perYear : ui.billing.perMonth}
                </p>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {ROWS.map((row) => (
            <tr key={row.label} className="border-b border-gray-100">
              <th className="py-3 pr-4 text-left text-sm font-medium text-gray-700" scope="row">
                {row.label}
              </th>
              {plans.map((plan) => {
                const value = row.value(plan);
                return (
                  <td key={plan.id} className="px-4 py-3 text-gray-900">
                    {value === null ? "—" : value.toLocaleString()}
                  </td>
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function formatCurrency(amount: number) {
  return new Intl.NumberFormat(undefined, {
    style: "currency",
    currency: "USD",
  }).format(amount);
}
