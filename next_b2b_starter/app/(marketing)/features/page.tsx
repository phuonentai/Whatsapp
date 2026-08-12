import type { Metadata } from "next";
import {
  BarChart3,
  Bot,
  Check,
  CreditCard,
  Database,
  FileText,
  Inbox,
  LayoutTemplate,
  Megaphone,
  ShieldCheck,
  type LucideIcon,
} from "lucide-react";
import { PageHero } from "@/components/marketing/page-hero";
import { CtaBanner } from "@/components/marketing/cta-banner";
import { Reveal } from "@/components/marketing/reveal";
import { cn } from "@/lib/utils";
import { copy } from "@/lib/copy/ui";

/** Distinct icon per section, in the same order as the copy `pageFeaturesItems` array. */
const SECTION_ICONS: LucideIcon[] = [
  Inbox,
  Bot,
  Database,
  FileText,
  CreditCard,
  Megaphone,
  LayoutTemplate,
  BarChart3,
  ShieldCheck,
];

export function generateMetadata(): Metadata {
  return {
    title: copy("marketing", "pageFeaturesTitle"),
    description: copy("marketing", "pageFeaturesLead"),
  };
}

export default function FeaturesPage() {
  const sections = copy("marketing", "pageFeaturesItems");

  return (
    <>
      <PageHero
        eyebrow={copy("marketing", "featuresEyebrow")}
        title={copy("marketing", "pageFeaturesTitle")}
        lead={copy("marketing", "pageFeaturesLead")}
      />
      {sections.map((section, index) => {
        const Icon = SECTION_ICONS[index % SECTION_ICONS.length];
        const dark = index % 2 === 1;
        return (
          <section
            key={section.title}
            className={cn(dark ? "bg-muted/50" : "bg-background", "py-16 lg:py-24")}
          >
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
              <div className="grid lg:grid-cols-2 gap-12 lg:gap-16 items-center">
                <div className={cn(dark && "lg:order-last")}>
                  <div
                    className={cn(
                      "w-14 h-14 rounded-xl flex items-center justify-center mb-6",
                      "bg-primary/10 text-primary"
                    )}
                  >
                    <Icon className="w-7 h-7" />
                  </div>
                  <h2
                    className={cn(
                      "font-heading text-2xl sm:text-3xl font-bold tracking-tight text-balance text-foreground"
                    )}
                  >
                    {section.title}
                  </h2>
                  <p
                    className={cn("mt-4 text-lg leading-relaxed text-muted-foreground")}
                  >
                    {section.description}
                  </p>
                </div>
                <ul className="grid sm:grid-cols-2 gap-4">
                  {section.bullets.map((bullet) => (
                    <li
                      key={bullet}
                      className={cn(
                        "rounded-xl border p-5 flex items-start gap-3",
                        dark
                          ? "bg-card border-border"
                          : "bg-card border-border"
                      )}
                    >
                      <Check
                        className={cn("w-5 h-5 mt-0.5 shrink-0 text-primary")}
                      />
                      <span
                        className={cn("text-sm leading-relaxed text-muted-foreground")}
                      >
                        {bullet}
                      </span>
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          </section>
        );
      })}
      <Reveal>
        <CtaBanner />
      </Reveal>
    </>
  );
}
