import { AlertCircle, CheckCircle2, XCircle, Sparkles } from "lucide-react";
import { SectionHeading } from "@/components/marketing/section-heading";
import { copy } from "@/lib/copy/ui";

export function Comparison() {
  const traditionalItems = copy("marketing", "comparisonTraditionalItems");
  const aiItems = copy("marketing", "comparisonAiItems");

  return (
    <section className="bg-background py-16 lg:py-24">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <SectionHeading
          title={copy("marketing", "comparisonTitle")}
          lead={copy("marketing", "comparisonLead")}
        />
        <div className="grid lg:grid-cols-2 gap-6">
          {/* Traditional card */}
          <div className="rounded-2xl border border-border bg-muted/50 p-8">
            <div className="flex items-center gap-3 mb-6">
              <div className="w-10 h-10 rounded-lg bg-red-500/10 flex items-center justify-center text-red-500 shrink-0">
                <XCircle className="w-5 h-5" />
              </div>
              <h3 className="font-heading text-xl font-bold text-foreground">
                {copy("marketing", "comparisonTraditional")}
              </h3>
            </div>
            <ul className="space-y-4">
              {traditionalItems.map((item) => (
                <li
                  key={item}
                  className="flex items-start gap-3 text-muted-foreground leading-relaxed"
                >
                  <AlertCircle className="w-5 h-5 text-red-500 shrink-0 mt-0.5" />
                  <span>{item}</span>
                </li>
              ))}
            </ul>
          </div>
          {/* NexoChat AI card */}
          <div className="relative overflow-hidden rounded-2xl border border-primary/30 bg-primary/5 p-8">
            <div className="absolute -top-16 -right-16 w-48 h-48 rounded-full bg-primary/10" />
            <div className="relative flex items-center gap-3 mb-6">
              <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center text-primary shrink-0">
                <Sparkles className="w-5 h-5" />
              </div>
              <h3 className="font-heading text-xl font-bold text-foreground">
                {copy("marketing", "comparisonWithAi")}
              </h3>
            </div>
            <ul className="relative space-y-4">
              {aiItems.map((item) => (
                <li
                  key={item}
                  className="flex items-start gap-3 text-foreground leading-relaxed"
                >
                  <CheckCircle2 className="w-5 h-5 text-primary shrink-0 mt-0.5" />
                  <span>{item}</span>
                </li>
              ))}
            </ul>
          </div>
        </div>
      </div>
    </section>
  );
}
