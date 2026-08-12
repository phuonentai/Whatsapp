import Link from "next/link";
import { format } from "date-fns";
import { es } from "date-fns/locale";
import { Badge } from "@/components/ui/badge";
import { copy, tpl } from "@/lib/copy/ui";
import type { BlogPostMeta } from "@/lib/content";

/** Parse "YYYY-MM-DD" as a local-time Date so formatters never shift the day (UTC vs local). */
function parseISODate(iso: string): Date {
  const [y, m, d] = iso.split("-").map(Number);
  return new Date(y, m - 1, d);
}

export function BlogCard({ post }: { post: BlogPostMeta }) {
  return (
    <Link
      href={`/blog/${post.slug}`}
      className="group flex h-full flex-col bg-card border border-border rounded-2xl p-6 transition hover:shadow-lg hover:shadow-slate-900/5 hover:border-primary/40"
    >
      <Badge
        variant="outline"
        className="self-start rounded-full border-primary/30 bg-primary/10 text-primary"
      >
        {post.category}
      </Badge>
      <h3 className="mt-4 font-heading text-lg font-bold text-foreground line-clamp-2 transition-colors group-hover:text-primary">
        {post.title}
      </h3>
      <p className="mt-2 text-sm text-muted-foreground line-clamp-2">{post.description}</p>
      <div className="mt-6 flex items-center justify-between border-t border-border pt-4 text-xs text-muted-foreground">
        <time dateTime={post.date}>
          {format(parseISODate(post.date), "d 'de' MMMM 'de' yyyy", { locale: es })}
        </time>
        <span>{tpl(copy("marketing", "blogReadTime"), { minutes: post.readingTimeMinutes })}</span>
      </div>
    </Link>
  );
}
