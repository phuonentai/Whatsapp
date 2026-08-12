import {
  BarChart3,
  Bot,
  CreditCard,
  FileText,
  Inbox,
  LayoutTemplate,
  Megaphone,
  ShieldCheck,
  type LucideIcon,
} from "lucide-react";
import { SectionHeading } from "@/components/marketing/section-heading";
import { copy } from "@/lib/copy/ui";

/** Distinct icon per feature, in the same order as the copy `features` array. */
const FEATURE_ICONS: LucideIcon[] = [
  Bot,
  FileText,
  CreditCard,
  Inbox,
  Megaphone,
  LayoutTemplate,
  BarChart3,
  ShieldCheck,
];

export function FeatureGrid() {
  const features = copy("marketing", "features");

  return (
    <section className="bg-card border-y border-border py-16 lg:py-24">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <SectionHeading
          eyebrow={copy("marketing", "featuresEyebrow")}
          title={copy("marketing", "featuresTitle")}
          lead={copy("marketing", "featuresLead")}
        />
        <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-6">
          {features.map((feature, index) => {
            const Icon = FEATURE_ICONS[index % FEATURE_ICONS.length];
            return (
              <div
                key={feature.title}
                className="group rounded-2xl border border-border bg-background p-6 transition-colors hover:border-primary/40"
              >
                <div className="w-11 h-11 rounded-lg bg-primary/10 text-primary flex items-center justify-center mb-4 transition-colors group-hover:bg-primary/15">
                  <Icon className="w-5 h-5" />
                </div>
                <h3 className="font-heading text-lg font-semibold text-foreground mb-2">
                  {feature.title}
                </h3>
                <p className="text-sm text-muted-foreground leading-relaxed">
                  {feature.description}
                </p>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}
