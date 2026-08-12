import type { Metadata } from "next";
import { PageHero } from "@/components/marketing/page-hero";
import { SectionHeading } from "@/components/marketing/section-heading";
import { CourseCard } from "@/components/marketing/course-card";
import { JsonLd } from "@/components/seo/jsonld";
import { copy } from "@/lib/copy/ui";
import { getCourse, getCourses, type Course } from "@/lib/content";

const SITE_URL = "https://yourdomain.com";

export function generateMetadata(): Metadata {
  return {
    title: copy("marketing", "academyTitle"),
    description: copy("marketing", "academyLead"),
  };
}

export default function AcademyPage() {
  const courses = getCourses()
    .map((course) => getCourse(course.slug))
    .filter((course): course is Course => course !== null);
  const tracks = [...new Set(courses.map((course) => course.meta.track))];

  return (
    <>
      <PageHero
        eyebrow={copy("marketing", "academyEyebrow")}
        title={copy("marketing", "academyTitle")}
        lead={copy("marketing", "academyLead")}
      />
      <section className="bg-muted/50 py-16 lg:py-24">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <SectionHeading
            eyebrow={copy("marketing", "academyEyebrow")}
            title={copy("marketing", "academyTracks")}
            align="left"
          />
          <div className="space-y-16">
            {tracks.map((track) => (
              <div key={track}>
                <h3 className="font-heading text-xl font-bold text-foreground tracking-tight">
                  {track}
                </h3>
                <div className="mt-6 grid gap-6 md:grid-cols-2 lg:grid-cols-3">
                  {courses
                    .filter((course) => course.meta.track === track)
                    .map((course) => (
                      <CourseCard key={course.meta.slug} course={course} />
                    ))}
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>
      <JsonLd
        id="academy-itemlist"
        data={{
          "@context": "https://schema.org",
          "@type": "ItemList",
          name: copy("marketing", "academyTitle"),
          itemListElement: courses.map((course, index) => ({
            "@type": "ListItem",
            position: index + 1,
            item: {
              "@type": "Course",
              name: course.meta.title,
              description: course.meta.description,
              url: `${SITE_URL}/academy/${course.meta.slug}`,
            },
          })),
        }}
      />
    </>
  );
}
