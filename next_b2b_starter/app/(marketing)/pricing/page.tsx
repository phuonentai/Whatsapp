import type { Metadata } from "next";
import { PageHero } from "@/components/marketing/page-hero";
import { PricingPlans } from "@/components/marketing/pricing";
import { copy } from "@/lib/copy/ui";

export function generateMetadata(): Metadata {
  return {
    title: copy("marketing", "pagePricingTitle"),
    description: copy("marketing", "pagePricingLead"),
  };
}

export default function PricingPage() {
  return (
    <>
      <PageHero
        eyebrow={copy("marketing", "pricingEyebrow")}
        title={copy("marketing", "pagePricingTitle")}
        lead={copy("marketing", "pagePricingLead")}
      />
      <PricingPlans />
    </>
  );
}
