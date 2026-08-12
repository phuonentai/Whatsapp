import type { Metadata } from "next";
import { PageHero } from "@/components/marketing/page-hero";
import { BlogFilters } from "@/components/marketing/blog-filters";
import { JsonLd } from "@/components/seo/jsonld";
import { copy } from "@/lib/copy/ui";
import { getBlogCategories, getBlogPosts } from "@/lib/content";

const SITE_URL = "https://yourdomain.com";

export function generateMetadata(): Metadata {
  return {
    title: copy("marketing", "blogTitle"),
    description: copy("marketing", "blogLead"),
  };
}

export default function BlogIndexPage() {
  const posts = getBlogPosts();
  const categories = getBlogCategories();

  return (
    <>
      <PageHero
        eyebrow={copy("marketing", "blogEyebrow")}
        title={copy("marketing", "blogTitle")}
        lead={copy("marketing", "blogLead")}
      />
      <section className="bg-white py-20 lg:py-28">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <BlogFilters posts={posts} categories={categories} />
        </div>
      </section>
      <JsonLd
        id="blog-jsonld"
        data={{
          "@context": "https://schema.org",
          "@type": "Blog",
          name: "NexoChat Blog",
          description: copy("marketing", "blogLead"),
          url: `${SITE_URL}/blog`,
          blogPost: posts.map((post) => ({
            "@type": "BlogPosting",
            headline: post.title,
            description: post.description,
            datePublished: post.date,
            dateModified: post.date,
            author: { "@type": "Person", name: post.author },
            url: `${SITE_URL}/blog/${post.slug}`,
          })),
        }}
      />
    </>
  );
}
