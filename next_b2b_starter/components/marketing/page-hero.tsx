import { cn } from "@/lib/utils";

interface PageHeroProps {
  eyebrow?: string;
  title: string;
  lead?: string;
  children?: React.ReactNode;
  className?: string;
}

/** Hero oscuro Verifika para subpáginas (features, pricing, blog, academy, legal). */
export function PageHero({ eyebrow, title, lead, children, className }: PageHeroProps) {
  return (
    <section className={cn("relative bg-slate-900 text-white overflow-hidden", className)}>
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_at_top_right,rgba(16,185,129,0.16),transparent_55%)]"
      />
      <div className="relative max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16 lg:py-20">
        <div className="max-w-3xl">
          {eyebrow && (
            <div className="inline-flex items-center gap-2 bg-emerald-500/10 border border-emerald-500/30 rounded-full px-4 py-2 mb-6">
              <span className="w-2 h-2 bg-emerald-400 rounded-full" />
              <span className="text-emerald-400 text-sm font-medium">{eyebrow}</span>
            </div>
          )}
          <h1 className="font-heading text-4xl sm:text-5xl font-bold tracking-tight text-balance">
            {title}
          </h1>
          {lead && (
            <p className="mt-6 text-lg text-slate-400 leading-relaxed max-w-2xl">
              {lead}
            </p>
          )}
          {children}
        </div>
      </div>
    </section>
  );
}
