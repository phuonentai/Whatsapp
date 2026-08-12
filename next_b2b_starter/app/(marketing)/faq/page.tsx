import type { Metadata } from "next";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { PageHero } from "@/components/marketing/page-hero";
import { JsonLd } from "@/components/seo/jsonld";
import { copy } from "@/lib/copy/ui";

export function generateMetadata(): Metadata {
  return {
    title: copy("marketing", "pageFaqTitle"),
    description: copy("marketing", "faqLead"),
  };
}

export default function FaqPage() {
  const faq = copy("marketing", "faq");

  const faqJsonLd = {
    "@context": "https://schema.org",
    "@type": "FAQPage",
    mainEntity: faq.map((item) => ({
      "@type": "Question",
      name: item.question,
      acceptedAnswer: {
        "@type": "Answer",
        text: item.answer,
      },
    })),
  };

  return (
    <>
      <PageHero
        eyebrow={copy("marketing", "navFaq")}
        title={copy("marketing", "pageFaqTitle")}
        lead={copy("marketing", "faqLead")}
      />
      <section className="bg-background py-16 lg:py-24">
        <JsonLd id="faq-page-jsonld" data={faqJsonLd} />
        <div className="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8">
          <Accordion type="single" collapsible className="space-y-4">
            {faq.map((item, index) => (
              <AccordionItem
                key={item.question}
                value={`faq-item-${index}`}
                className="rounded-xl border border-border bg-card px-5"
              >
                <AccordionTrigger className="text-base font-semibold text-foreground hover:no-underline [&[data-state=open]]:text-primary">
                  {item.question}
                </AccordionTrigger>
                <AccordionContent className="text-muted-foreground leading-relaxed">
                  {item.answer}
                </AccordionContent>
              </AccordionItem>
            ))}
          </Accordion>
        </div>
      </section>
    </>
  );
}
