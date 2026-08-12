"use client";

import { useState } from "react";
import Link from "next/link";
import { Check } from "lucide-react";
import { Button } from "@/components/ui/button";
import { JsonLd } from "@/components/seo/jsonld";
import { copy } from "@/lib/copy/ui";
import { cn } from "@/lib/utils";

const SITE_URL = "https://yourdomain.com";

type BillingInterval = "monthly" | "yearly";

interface Plan {
  name: string;
  price: string;
  suffix: string;
  currency: "COP" | "USD";
  features: readonly string[];
  cta: string;
  ctaHref: string;
  popular?: boolean;
}

const PLANS: Plan[] = [
  {
    name: copy("marketing", "pricingFreeName"),
    price: copy("marketing", "pricingFreePrice"),
    suffix: copy("marketing", "pricingPerMonth"),
    currency: "COP",
    features: copy("marketing", "pricingFreeFeatures"),
    cta: copy("marketing", "pricingCtaStart"),
    ctaHref: "/signup",
  },
  {
    name: copy("marketing", "pricingStarterName"),
    price: copy("marketing", "pricingStarterPriceUsd"),
    suffix: copy("marketing", "pricingPerMonthUsd"),
    currency: "USD",
    features: copy("marketing", "pricingStarterFeatures"),
    cta: copy("marketing", "pricingCtaTry"),
    ctaHref: "/signup",
  },
  {
    name: copy("marketing", "pricingProName"),
    price: copy("marketing", "pricingProPriceUsd"),
    suffix: copy("marketing", "pricingPerMonthUsd"),
    currency: "USD",
    features: copy("marketing", "pricingProFeatures"),
    cta: copy("marketing", "pricingCtaStart"),
    ctaHref: "/signup",
    popular: true,
  },
  {
    name: copy("marketing", "pricingEnterpriseName"),
    price: copy("marketing", "pricingEnterprisePrice"),
    suffix: "",
    currency: "USD",
    features: copy("marketing", "pricingEnterpriseFeatures"),
    cta: copy("marketing", "pricingCtaContact"),
    ctaHref: "mailto:ventas@nexochat.co",
  },
];

/**
 * Annual price = monthly price × 0.8, rounded. The monthly value is derived
 * from the copy price string (e.g. "$39" → 39 → "$31") so copy stays the
 * single source of truth. Returns null for non-numeric prices ("Custom").
 */
function annualPrice(monthly: string): string | null {
  const digits = monthly.replace(/\D/g, "");
  if (!digits) return null;
  return `$${Math.round(Number(digits) * 0.8)}`;
}

export function PricingPlans() {
  const [interval, setInterval] = useState<BillingInterval>("monthly");
  const yearly = interval === "yearly";

  return (
    <section className="bg-slate-50 py-20 lg:py-28">
      <JsonLd
        id="pricing-jsonld"
        data={{
          "@context": "https://schema.org",
          "@type": "OfferCatalog",
          name: "NexoChat Planes y Precios",
          description: copy("marketing", "pricingLead"),
          url: `${SITE_URL}/pricing`,
          itemListElement: PLANS.filter((plan) => annualPrice(plan.price) !== null).map(
            (plan) => ({
              "@type": "Product",
              name: plan.name,
              offers: {
                "@type": "Offer",
                price: plan.price.replace(/\D/g, ""),
                priceCurrency: plan.currency,
                availability: "https://schema.org/InStock",
                url: `${SITE_URL}/signup`,
              },
            })
          ),
        }}
      />
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-center">
          <div
            role="group"
            aria-label="Facturación"
            className="inline-flex items-center gap-1 rounded-full border border-border bg-card p-1 shadow-sm"
          >
            <button
              type="button"
              aria-pressed={!yearly}
              onClick={() => setInterval("monthly")}
              className={cn(
                "rounded-full px-5 py-2 text-sm font-semibold transition-all",
                !yearly
                  ? "bg-primary text-primary-foreground shadow"
                  : "text-muted-foreground hover:text-foreground"
              )}
            >
              {copy("marketing", "pricingMonthly")}
            </button>
            <button
              type="button"
              aria-pressed={yearly}
              onClick={() => setInterval("yearly")}
              className={cn(
                "inline-flex items-center gap-2 rounded-full px-5 py-2 text-sm font-semibold transition-all",
                yearly
                  ? "bg-primary text-primary-foreground shadow"
                  : "text-muted-foreground hover:text-foreground"
              )}
            >
              {copy("marketing", "pricingYearly")}
              <span
                className={cn(
                  "rounded-full border px-2 py-0.5 text-xs font-bold",
                  yearly
                    ? "bg-primary border-primary text-primary-foreground"
                    : "bg-primary/10 border-primary/20 text-primary"
                )}
              >
                {copy("marketing", "pricingYearlyDiscount")}
              </span>
            </button>
          </div>
        </div>

        <div className="mt-12 grid gap-6 md:grid-cols-2 lg:grid-cols-4 items-stretch">
          {PLANS.map((plan) => {
            const annual = annualPrice(plan.price);
            const displayed = yearly && annual ? annual : plan.price;
            return (
              <div
                key={plan.name}
                className={cn(
                  "relative rounded-2xl border p-8 flex flex-col",
                  plan.popular
                    ? "bg-card border-2 border-primary shadow-lg shadow-primary/10"
                    : "bg-card border-border"
                )}
              >
                {plan.popular && (
                  <span className="absolute -top-3 left-1/2 -translate-x-1/2 bg-primary text-primary-foreground text-xs font-bold px-3 py-1 rounded-full whitespace-nowrap">
                    {copy("marketing", "pricingPopular")}
                  </span>
                )}
                <h3 className="font-heading text-lg font-semibold text-foreground">{plan.name}</h3>
                <div className="mt-4 flex items-baseline gap-1.5">
                  <span className="text-3xl font-bold tracking-tight tabular-nums text-foreground">
                    {displayed}
                  </span>
                  {plan.suffix && (
                    <span className="text-sm text-muted-foreground">
                      {plan.suffix}
                    </span>
                  )}
                </div>
                <ul className="mt-6 space-y-3 flex-1">
                  {plan.features.map((feature) => (
                    <li
                      key={feature}
                      className="flex items-start gap-2 text-sm leading-relaxed text-muted-foreground"
                    >
                      <Check className="w-4 h-4 mt-0.5 shrink-0 text-primary" />
                      <span>{feature}</span>
                    </li>
                  ))}
                </ul>
                <Link href={plan.ctaHref} className="mt-8 block">
                  <Button
                    size="lg"
                    className={cn(
                      "w-full rounded-xl font-semibold",
                      plan.popular
                        ? "bg-primary hover:bg-primary/90 text-primary-foreground"
                        : "bg-transparent border border-border text-foreground hover:border-primary hover:text-primary"
                    )}
                  >
                    {plan.cta}
                  </Button>
                </Link>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}
