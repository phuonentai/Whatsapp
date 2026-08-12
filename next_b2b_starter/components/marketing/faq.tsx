import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { JsonLd } from "@/components/seo/jsonld";
import { SectionHeading } from "@/components/marketing/section-heading";
import { copy } from "@/lib/copy/ui";

export function Faq() {
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
    <section className="bg-white py-20 lg:py-28">
      <JsonLd id="faq-jsonld" data={faqJsonLd} />
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <SectionHeading
          title={copy("marketing", "faqTitle")}
          lead={copy("marketing", "faqLead")}
        />
        <div className="max-w-3xl mx-auto">
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
      </div>
    </section>
  );
}
