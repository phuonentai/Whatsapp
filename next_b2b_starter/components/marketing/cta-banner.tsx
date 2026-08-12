import Link from "next/link";
import { ArrowRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { copy } from "@/lib/copy/ui";

export function CtaBanner() {
  return (
    <section className="bg-background py-16 lg:py-24">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="relative overflow-hidden rounded-3xl bg-primary px-8 py-16 lg:px-16 lg:py-20 text-center">
          <div className="absolute -top-24 -left-24 w-64 h-64 rounded-full bg-primary-foreground/10" />
          <div className="absolute -bottom-24 -right-24 w-64 h-64 rounded-full bg-primary-foreground/10" />
          <div className="relative max-w-2xl mx-auto">
            <h2 className="font-heading text-3xl sm:text-4xl font-bold text-primary-foreground tracking-tight text-balance">
              {copy("marketing", "ctaTitle")}
            </h2>
            <p className="mt-4 text-lg text-primary-foreground/80 leading-relaxed">
              {copy("marketing", "ctaLead")}
            </p>
            <div className="mt-8 flex flex-col sm:flex-row items-center justify-center gap-4">
              <Link href="/signup">
                <Button
                  size="lg"
                  className="bg-white text-primary hover:bg-white/90 rounded-xl font-semibold"
                >
                  {copy("marketing", "ctaButton")}
                  <ArrowRight className="w-5 h-5" />
                </Button>
              </Link>
              <Link href="/pricing">
                <Button
                  size="lg"
                  variant="outline"
                  className="bg-transparent border-primary-foreground/40 text-primary-foreground hover:border-primary-foreground hover:bg-primary-foreground/10 rounded-xl font-semibold"
                >
                  {copy("marketing", "ctaSecondary")}
                </Button>
              </Link>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
