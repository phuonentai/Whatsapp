export interface PolarPlan {
  id: string;
  name: string;
  description: string | null;
  price: number; // USD per month
  interval: "month" | "year";
  productId: string;
  priceId: string | null;
  includedSeats: number | null;
  includedInvoices: number | null;
  benefits: string[];
  badge?: string;
  metadata?: Record<string, unknown>;
}

/**
 * Plans are now fetched dynamically from Polar via /api/billing/products
 * Use the useProductsQuery() hook to fetch products in React components
 */

export function getPlanById(
  plans: PolarPlan[],
  planId: string | null | undefined
): PolarPlan | null {
  if (!planId) return null;
  return plans.find((plan) => plan.id === planId) ?? null;
}

export function getPlanByProductId(
  plans: PolarPlan[],
  productId: string | null | undefined
): PolarPlan | null {
  if (!productId) return null;
  return plans.find((plan) => plan.productId === productId) ?? null;
}

export function getDefaultPlan(plans: PolarPlan[]): PolarPlan | null {
  return plans[0] ?? null;
}

/**
 * Extract plan information from Polar product metadata
 *
 * Polar now exposes product benefits and metadata directly.
 * This function attempts to extract plan info from the product object
 * and falls back to static plans if not available.
 */
export function getPlanFromProduct(product: {
  id: string;
  name: string;
  metadata?: Record<string, unknown> | null;
  benefits?: Array<{ description: string }> | null;
  prices?: Array<{
    id?: string;
    recurringInterval?: string;
    amount?: number;
    currency?: string;
  }> | null;
}): Partial<PolarPlan> | null {
  if (!product) {
    return null;
  }

  const metadata = product.metadata ?? {};
  const benefits = product.benefits?.map((b) => b.description) ?? [];
  const price = product.prices?.[0];

  if (!price || !price.id) {
    return null;
  }

  // Try to extract plan ID from metadata
  const planId = typeof metadata.plan_id === "string" ? metadata.plan_id : null;

  // Extract seats and invoices from metadata if available
  const includedSeats =
    typeof metadata.included_seats === "number"
      ? metadata.included_seats
      : typeof metadata.seats === "number"
        ? metadata.seats
        : undefined;

  const includedInvoices =
    typeof metadata.included_invoices === "number"
      ? metadata.included_invoices
      : typeof metadata.invoices === "number"
        ? metadata.invoices
        : undefined;

  console.info("[Polar] Extracted plan from product metadata", {
    productId: product.id,
    productName: product.name,
    planId,
    includedSeats,
    includedInvoices,
    benefits: benefits.length,
    hasPrice: Boolean(price),
  });

  return {
    id: planId ?? product.id,
    name: product.name,
    productId: product.id,
    priceId: price?.id ?? null,
    benefits,
    includedSeats,
    includedInvoices,
    price: price.amount ? price.amount / 100 : undefined, // Convert cents to dollars
    interval: (price.recurringInterval as "month" | undefined) ?? "month",
  };
}
