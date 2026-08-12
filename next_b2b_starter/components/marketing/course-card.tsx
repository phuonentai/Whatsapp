import Link from "next/link";
import { Badge } from "@/components/ui/badge";
import { copy, tpl } from "@/lib/copy/ui";
import type { Course } from "@/lib/content";

export function CourseCard({ course }: { course: Course }) {
  const { meta, lessons } = course;
  return (
    <Link
      href={`/academy/${meta.slug}`}
      className="group flex h-full flex-col bg-card text-foreground rounded-2xl p-6 border border-border transition hover:border-primary/40 hover:shadow-lg hover:shadow-slate-900/5"
    >
      <Badge
        variant="outline"
        className="self-start rounded-full border-primary/30 bg-primary/10 text-primary"
      >
        {meta.level}
      </Badge>
      <h3 className="mt-4 font-heading text-lg font-bold text-foreground line-clamp-2 transition-colors group-hover:text-primary">
        {meta.title}
      </h3>
      <p className="mt-2 text-sm text-muted-foreground line-clamp-2">{meta.description}</p>
      <div className="mt-6 flex items-center justify-between border-t border-border pt-4 text-xs text-muted-foreground">
        <span>{tpl(copy("marketing", "academyLessonsCount"), { n: lessons.length })}</span>
        <span>{meta.durationMinutes} min</span>
      </div>
    </Link>
  );
}
