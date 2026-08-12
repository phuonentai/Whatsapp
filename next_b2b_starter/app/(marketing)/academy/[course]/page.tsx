import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { ArrowRight } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { PageHero } from "@/components/marketing/page-hero";
import { JsonLd } from "@/components/seo/jsonld";
import { copy, tpl } from "@/lib/copy/ui";
import { getCourse, getCourses } from "@/lib/content";

const SITE_URL = "https://yourdomain.com";

interface CoursePageProps {
  params: Promise<{ course: string }>;
}

export const dynamicParams = false;


export async function generateMetadata({ params }: CoursePageProps): Promise<Metadata> {
  const { course: courseSlug } = await params;
  const course = getCourse(courseSlug);
  if (!course) return {};
  return {
    title: course.meta.title,
    description: course.meta.description,
  };
}

export default async function CoursePage({ params }: CoursePageProps) {
  const { course: courseSlug } = await params;
  const course = getCourse(courseSlug);
  if (!course) notFound();

  const { meta, lessons } = course;
  const firstLesson = lessons[0];

  return (
    <>
      <PageHero eyebrow={copy("marketing", "academyCourse")} title={meta.title} lead={meta.description}>
        <div className="mt-8 flex flex-wrap items-center gap-x-5 gap-y-3 text-sm text-slate-300">
          <Badge
            variant="outline"
            className="rounded-full border-primary/30 bg-primary/10 text-primary"
          >
            {copy("marketing", "academyLevel")}: {meta.level}
          </Badge>
          <span>
            {copy("marketing", "academyDuration")}: {meta.durationMinutes} min
          </span>
          <span aria-hidden="true">·</span>
          <span>{tpl(copy("marketing", "academyLessonsCount"), { n: lessons.length })}</span>
        </div>
        {firstLesson && (
          <div className="mt-8">
            <Link href={`/academy/${meta.slug}/${firstLesson.slug}`}>
              <Button
                size="lg"
                className="bg-primary hover:bg-primary/90 text-primary-foreground rounded-xl font-bold transition-all transform "
              >
                {copy("marketing", "academyStartCourse")}
                <ArrowRight className="w-5 h-5" />
              </Button>
            </Link>
          </div>
        )}
      </PageHero>

      <section className="bg-muted/50 py-16 lg:py-24">
        <div className="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8">
          <ol className="space-y-4">
            {lessons.map((lesson, index) => (
              <li key={lesson.slug}>
                <Link
                  href={`/academy/${meta.slug}/${lesson.slug}`}
                  className="group flex items-start gap-4 rounded-2xl bg-card border border-border p-5 transition hover:border-primary/40 hover:shadow-lg"
                >
                  <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-primary/10 text-sm font-bold text-primary">
                    {index + 1}
                  </span>
                  <span className="min-w-0 flex-1">
                    <span className="block font-heading font-bold text-foreground transition-colors group-hover:text-primary">
                      {lesson.title}
                    </span>
                    <span className="mt-1 block text-sm text-muted-foreground line-clamp-2">
                      {lesson.description}
                    </span>
                  </span>
                  <span className="shrink-0 text-xs text-muted-foreground">
                    {lesson.durationMinutes} min
                  </span>
                </Link>
              </li>
            ))}
          </ol>
        </div>
      </section>

      <JsonLd
        id={`course-${meta.slug}`}
        data={{
          "@context": "https://schema.org",
          "@type": "Course",
          name: meta.title,
          description: meta.description,
          provider: { "@type": "Organization", name: "NexoChat", url: SITE_URL },
          hasCourseInstance: {
            "@type": "CourseInstance",
            courseMode: "online",
            courseWorkload: `PT${meta.durationMinutes}M`,
          },
          url: `${SITE_URL}/academy/${meta.slug}`,
        }}
      />
    </>
  );
}
