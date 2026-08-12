import Link from "next/link";
import { ArrowLeft, ArrowRight } from "lucide-react";
import { copy } from "@/lib/copy/ui";
import type { LessonMeta } from "@/lib/content";

interface LessonNavProps {
  courseSlug: string;
  prev: LessonMeta | null;
  next: LessonMeta | null;
}

export function LessonNav({ courseSlug, prev, next }: LessonNavProps) {
  return (
    <nav aria-label="Navegación entre lecciones" className="mt-12 grid gap-4 sm:grid-cols-2">
      {prev ? (
        <Link
          href={`/academy/${courseSlug}/${prev.slug}`}
          className="group flex flex-col rounded-2xl bg-card border border-border p-5 transition hover:border-primary/40 hover:shadow-lg hover:shadow-slate-900/5"
        >
          <span className="inline-flex items-center gap-2 text-sm font-medium text-primary">
            <ArrowLeft className="h-4 w-4 transition-transform group-hover:-translate-x-0.5" />
            {copy("marketing", "academyPrevious")}
          </span>
          <span className="mt-2 font-heading font-semibold text-foreground line-clamp-2 transition-colors group-hover:text-primary">
            {prev.title}
          </span>
        </Link>
      ) : (
        <div aria-hidden="true" />
      )}
      {next ? (
        <Link
          href={`/academy/${courseSlug}/${next.slug}`}
          className="group flex flex-col rounded-2xl bg-card border border-border p-5 text-right transition hover:border-primary/40 hover:shadow-lg hover:shadow-slate-900/5"
        >
          <span className="inline-flex items-center justify-end gap-2 text-sm font-medium text-primary">
            {copy("marketing", "academyNext")}
            <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
          </span>
          <span className="mt-2 font-heading font-semibold text-foreground line-clamp-2 transition-colors group-hover:text-primary">
            {next.title}
          </span>
        </Link>
      ) : (
        <div aria-hidden="true" />
      )}
    </nav>
  );
}
