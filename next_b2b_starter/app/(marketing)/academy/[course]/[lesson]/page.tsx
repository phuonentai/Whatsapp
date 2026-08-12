import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { ArrowLeft } from "lucide-react";
import { LessonNav } from "@/components/marketing/lesson-nav";
import { Prose } from "@/components/marketing/prose";
import { JsonLd } from "@/components/seo/jsonld";
import { copy } from "@/lib/copy/ui";
import { getCourse, getCourses, getLesson } from "@/lib/content";

const SITE_URL = "https://yourdomain.com";

interface LessonPageProps {
  params: Promise<{ course: string; lesson: string }>;
}

export const dynamicParams = false;


export async function generateMetadata({ params }: LessonPageProps): Promise<Metadata> {
  const { course: courseSlug, lesson: lessonSlug } = await params;
  const lesson = getLesson(courseSlug, lessonSlug);
  if (!lesson) return {};
  return {
    title: lesson.title,
    description: lesson.description,
  };
}

export default async function LessonPage({ params }: LessonPageProps) {
  const { course: courseSlug, lesson: lessonSlug } = await params;
  const course = getCourse(courseSlug);
  if (!course) notFound();
  const lesson = getLesson(courseSlug, lessonSlug);
  if (!lesson) notFound();

  return (
    <>
      <article className="mx-auto max-w-3xl px-4 sm:px-6 py-16">
        <header className="rounded-2xl bg-card border border-border p-8 sm:p-10">
          <p className="text-sm font-semibold uppercase tracking-wider text-primary">
            {course.meta.title}
          </p>
          <h1 className="mt-3 font-heading text-3xl font-bold tracking-tight text-balance sm:text-4xl">
            {lesson.title}
          </h1>
          <p className="mt-4 text-muted-foreground leading-relaxed">{lesson.description}</p>
          <div className="mt-6 text-sm text-muted-foreground">{lesson.durationMinutes} min</div>
        </header>

        <Prose content={lesson.body} className="mt-10" />

        <LessonNav courseSlug={courseSlug} prev={lesson.prev} next={lesson.next} />

        <div className="mt-10 flex flex-wrap items-center gap-x-6 gap-y-3 text-sm font-medium">
          <Link
            href={`/academy/${courseSlug}`}
            className="inline-flex items-center gap-2 text-primary transition-colors hover:text-primary"
          >
            <ArrowLeft className="h-4 w-4" />
            {copy("marketing", "academyBackToCourse")}
          </Link>
          <Link
            href="/academy"
            className="inline-flex items-center gap-2 text-muted-foreground transition-colors hover:text-foreground"
          >
            <ArrowLeft className="h-4 w-4" />
            {copy("marketing", "academyBackToAcademy")}
          </Link>
        </div>
      </article>

      <JsonLd
        id={`lesson-${courseSlug}-${lesson.slug}`}
        data={{
          "@context": "https://schema.org",
          "@type": "LearningResource",
          name: lesson.title,
          description: lesson.description,
          inLanguage: "es-CO",
          isPartOf: {
            "@type": "Course",
            name: course.meta.title,
            url: `${SITE_URL}/academy/${courseSlug}`,
          },
          timeRequired: `PT${lesson.durationMinutes}M`,
          educationalLevel: course.meta.level,
          url: `${SITE_URL}/academy/${courseSlug}/${lesson.slug}`,
        }}
      />
    </>
  );
}
