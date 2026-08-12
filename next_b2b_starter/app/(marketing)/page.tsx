import type { Metadata } from "next";
import Link from "next/link";
import { ArrowRight, Check } from "lucide-react";
import { Hero } from "@/components/marketing/hero";
import { Comparison } from "@/components/marketing/comparison";
import { FeatureGrid } from "@/components/marketing/feature-grid";
import { RoiCalculator } from "@/components/marketing/roi-calculator";
import { Faq } from "@/components/marketing/faq";
import { CtaBanner } from "@/components/marketing/cta-banner";
import { Reveal } from "@/components/marketing/reveal";
import { SectionHeading } from "@/components/marketing/section-heading";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { copy } from "@/lib/copy/ui";

export function generateMetadata(): Metadata {
  return {
    title: "CRM para WhatsApp con IA | Ventas y facturación automatizadas",
    description: copy("marketing", "heroLead"),
  };
}

interface Plan {
  name: string;
  price: string;
  perMonth: boolean;
  features: readonly string[];
  cta: string;
  popular?: boolean;
}

const PLANS: Plan[] = [
  {
    name: copy("marketing", "pricingFreeName"),
    price: copy("marketing", "pricingFreePrice"),
    perMonth: false,
    features: copy("marketing", "pricingFreeFeatures"),
    cta: copy("marketing", "pricingCtaStart"),
  },
  {
    name: copy("marketing", "pricingStarterName"),
    price: copy("marketing", "pricingStarterPrice"),
    perMonth: true,
    features: copy("marketing", "pricingStarterFeatures"),
    cta: copy("marketing", "pricingCtaTry"),
  },
  {
    name: copy("marketing", "pricingProName"),
    price: copy("marketing", "pricingProPrice"),
    perMonth: true,
    features: copy("marketing", "pricingProFeatures"),
    cta: copy("marketing", "pricingCtaStart"),
    popular: true,
  },
  {
    name: copy("marketing", "pricingEnterpriseName"),
    price: copy("marketing", "pricingEnterprisePrice"),
    perMonth: false,
    features: copy("marketing", "pricingEnterpriseFeatures"),
    cta: copy("marketing", "pricingCtaContact"),
  },
];

function PricingPreview() {
  return (
    <section className="bg-muted/50 py-16 lg:py-24">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <SectionHeading
          eyebrow={copy("marketing", "pricingEyebrow")}
          title={copy("marketing", "pricingTitle")}
          lead={copy("marketing", "pricingLead")}
        />
        <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-6 items-stretch">
          {PLANS.map((plan) => (
            <div
              key={plan.name}
              className={cn(
                "relative rounded-2xl border p-8 flex flex-col",
                plan.popular
                  ? "bg-card border-primary shadow-lg shadow-primary/10 lg:-mt-4 lg:mb-4"
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
                  {plan.price}
                </span>
                {plan.perMonth && (
                  <span className="text-sm text-muted-foreground">
                    {copy("marketing", "pricingPerMonth")}
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
              <Link href="/signup" className="mt-8 block">
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
          ))}
        </div>
        <div className="mt-12 text-center">
          <Link
            href="/pricing"
            className="inline-flex items-center gap-2 font-semibold text-primary hover:text-primary/80 transition-colors"
          >
            {copy("marketing", "pricingViewAll")}
            <ArrowRight className="w-4 h-4" />
          </Link>
        </div>
      </div>
    </section>
  );
}

export default function MarketingHomePage() {
  return (
    <>
      <Hero />
      <Reveal>
        <FeatureGrid />
      </Reveal>
      <Reveal>
        <Comparison />
      </Reveal>
      <Reveal>
        <RoiCalculator />
      </Reveal>
      <Reveal>
        <PricingPreview />
      </Reveal>
      <Reveal>
        <Faq />
      </Reveal>
      <Reveal>
        <CtaBanner />
      </Reveal>
    </>
  );
}
