import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { ArrowLeft } from "lucide-react";
import { format } from "date-fns";
import { es } from "date-fns/locale";
import { Badge } from "@/components/ui/badge";
import { Prose } from "@/components/marketing/prose";
import { JsonLd } from "@/components/seo/jsonld";
import { copy, tpl } from "@/lib/copy/ui";
import { getBlogPost, getBlogPosts } from "@/lib/content";

const SITE_URL = "https://yourdomain.com";

interface BlogArticlePageProps {
  params: Promise<{ slug: string }>;
}

export const dynamicParams = false;


export async function generateMetadata({
  params,
}: BlogArticlePageProps): Promise<Metadata> {
  const { slug } = await params;
  const post = getBlogPost(slug);
  if (!post) return {};
  return {
    title: post.meta.title,
    description: post.meta.description,
    openGraph: {
      title: post.meta.title,
      description: post.meta.description,
      type: "article",
      publishedTime: post.meta.date,
      authors: [post.meta.author],
    },
  };
}

/** Parse "YYYY-MM-DD" as a local-time Date so formatters never shift the day (UTC vs local). */
function parseISODate(iso: string): Date {
  const [y, m, d] = iso.split("-").map(Number);
  return new Date(y, m - 1, d);
}

export default async function BlogArticlePage({ params }: BlogArticlePageProps) {
  const { slug } = await params;
  const post = getBlogPost(slug);
  if (!post) notFound();

  const { meta } = post;

  return (
    <>
      <article className="max-w-3xl mx-auto px-4 sm:px-6 py-16">
        <header className="rounded-2xl bg-card border border-border p-8 sm:p-10">
          <Badge
            variant="outline"
            className="rounded-full border-primary/30 bg-primary/10 text-primary"
          >
            {meta.category}
          </Badge>
          <h1 className="mt-5 font-heading text-3xl font-bold tracking-tight text-balance sm:text-4xl">
            {meta.title}
          </h1>
          <div className="mt-6 flex flex-wrap items-center gap-x-5 gap-y-2 text-sm text-muted-foreground">
            <span>
              {copy("marketing", "blogBy")} {meta.author}
            </span>
            <span aria-hidden="true">·</span>
            <time dateTime={meta.date}>
              {format(parseISODate(meta.date), "d 'de' MMMM 'de' yyyy", { locale: es })}
            </time>
            <span aria-hidden="true">·</span>
            <span>
              {tpl(copy("marketing", "blogReadTime"), { minutes: meta.readingTimeMinutes })}
            </span>
          </div>
        </header>

        <Prose content={post.body} className="mt-10" />

        <Link
          href="/blog"
          className="mt-12 inline-flex items-center gap-2 font-medium text-primary transition-colors hover:text-primary"
        >
          <ArrowLeft className="h-4 w-4" />
          {copy("marketing", "blogBackToBlog")}
        </Link>
      </article>

      <JsonLd
        id={`blogposting-${meta.slug}`}
        data={{
          "@context": "https://schema.org",
          "@type": "BlogPosting",
          headline: meta.title,
          description: meta.description,
          datePublished: meta.date,
          dateModified: meta.date,
          author: { "@type": "Person", name: meta.author },
          publisher: {
            "@type": "Organization",
            name: "NexoChat",
            url: SITE_URL,
          },
          url: `${SITE_URL}/blog/${meta.slug}`,
          mainEntityOfPage: `${SITE_URL}/blog/${meta.slug}`,
        }}
      />
    </>
  );
}
