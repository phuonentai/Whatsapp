"use client";

import { useMemo, useState } from "react";
import { BlogCard } from "@/components/marketing/blog-card";
import { copy } from "@/lib/copy/ui";
import { cn } from "@/lib/utils";
import type { BlogPostMeta } from "@/lib/content";

interface BlogFiltersProps {
  posts: BlogPostMeta[];
  categories: string[];
}

/** Client-side category filter chips + post grid for the blog index. */
export function BlogFilters({ posts, categories }: BlogFiltersProps) {
  const all = copy("marketing", "blogAllCategories");
  const chips = [all, ...categories];
  const [active, setActive] = useState<string>(all);

  const visible = useMemo(
    () => (active === all ? posts : posts.filter((p) => p.category === active)),
    [posts, active, all]
  );

  return (
    <div>
      <div role="group" aria-label="Filtrar por categoría" className="flex flex-wrap gap-3">
        {chips.map((category) => {
          const isActive = category === active;
          return (
            <button
              key={category}
              type="button"
              onClick={() => setActive(category)}
              aria-pressed={isActive}
              className={cn(
                "rounded-full border px-4 py-2 text-sm font-medium transition",
                isActive
                  ? "border-primary bg-primary text-primary-foreground"
                  : "border-border bg-card text-muted-foreground hover:border-primary/40 hover:text-primary"
              )}
            >
              {category}
            </button>
          );
        })}
      </div>
      <div className="mt-10 grid gap-6 md:grid-cols-2 lg:grid-cols-3">
        {visible.map((post) => (
          <BlogCard key={post.slug} post={post} />
        ))}
      </div>
    </div>
  );
}
