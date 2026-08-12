import type { Metadata } from "next";
import { CalendarClock, Clock, Headset, MessageCircle, Rocket, ShieldCheck, Video } from "lucide-react";
import { PageHero } from "@/components/marketing/page-hero";
import { OnboardingInfoChecklist } from "@/components/marketing/onboarding-info-checklist";
import { SectionHeading } from "@/components/marketing/section-heading";
import { CtaBanner } from "@/components/marketing/cta-banner";
import { JsonLd } from "@/components/seo/jsonld";
import { copy } from "@/lib/copy/ui";

export function generateMetadata(): Metadata {
  return {
    title: "Onboarding guiado",
    description: copy("marketing", "onboardingInfoLead"),
  };
}

const PROCESS_STEPS = [
  {
    icon: MessageCircle,
    title: "Conexión de WhatsApp Business",
    body: "Migra tu número actual o configura uno nuevo con la API oficial de Meta, con nuestro acompañamiento.",
  },
  {
    icon: Rocket,
    title: "Entrenamiento de la IA",
    body: "Sube tu historial o lista de precios y el Copiloto aprende tus productos, precios y tono en menos de 60 segundos.",
  },
  {
    icon: ShieldCheck,
    title: "Facturación DIAN y cobros",
    body: "Conectamos Siigo y los medios de pago (PSE, Nequi, tarjetas) para facturar y cobrar dentro del chat.",
  },
  {
    icon: Headset,
    title: "Equipo y bandeja",
    body: "Configuramos asesores, roles y permisos para que todo tu equipo trabaje en una sola bandeja.",
  },
];

const SCHEDULE = [
  { day: "Día 1", text: "Kickoff y conexión de cuentas" },
  { day: "Día 2", text: "Entrenamiento de la IA y pruebas" },
  { day: "Día 3", text: "Activación de facturación y cobros" },
];

export default function OnboardingInfoPage() {
  const faq = copy("marketing", "onboardingInfoFaq");

  const faqJsonLd = {
    "@context": "https://schema.org",
    "@type": "FAQPage",
    mainEntity: faq.map((item) => ({
      "@type": "Question",
      name: item.question,
      acceptedAnswer: { "@type": "Answer", text: item.answer },
    })),
  };

  return (
    <>
      <PageHero
        eyebrow={copy("marketing", "onboardingInfoEyebrow")}
        title={copy("marketing", "onboardingInfoTitle")}
        lead={copy("marketing", "onboardingInfoLead")}
      />

      {/* Activación en menos de 24 horas */}
      <section className="bg-background py-16 lg:py-24">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <SectionHeading
            title={copy("marketing", "onboardingInfoActivationTitle")}
            lead={copy("marketing", "onboardingInfoActivationBody")}
          />
          <div className="grid md:grid-cols-3 gap-6">
            <div className="rounded-2xl border border-slate-200 bg-white p-8 text-center">
              <div className="mx-auto w-12 h-12 rounded-xl bg-emerald-500/10 text-emerald-600 flex items-center justify-center mb-4">
                <Clock className="w-6 h-6" />
              </div>
              <p className="text-3xl font-bold text-slate-900 tabular-nums">24h</p>
              <p className="text-sm text-slate-500 mt-1">Activación promedio</p>
            </div>
            <div className="rounded-2xl border border-slate-200 bg-white p-8 text-center">
              <div className="mx-auto w-12 h-12 rounded-xl bg-emerald-500/10 text-emerald-600 flex items-center justify-center mb-4">
                <Video className="w-6 h-6" />
              </div>
              <p className="text-3xl font-bold text-slate-900">Sesión en vivo</p>
              <p className="text-sm text-slate-500 mt-1">Incluida en tu plan</p>
            </div>
            <div className="rounded-2xl border border-slate-200 bg-white p-8 text-center">
              <div className="mx-auto w-12 h-12 rounded-xl bg-emerald-500/10 text-emerald-600 flex items-center justify-center mb-4">
                <CalendarClock className="w-6 h-6" />
              </div>
              <p className="text-3xl font-bold text-slate-900">Guía 1:1</p>
              <p className="text-sm text-slate-500 mt-1">Acompañamiento en cada etapa</p>
            </div>
          </div>
        </div>
      </section>

      {/* Proceso paso a paso */}
      <section className="bg-white border-y border-slate-200 py-16 lg:py-24">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <SectionHeading
            eyebrow={copy("marketing", "onboardingInfoStepsTitle")}
            title={copy("marketing", "onboardingInfoProcessTitle")}
            lead={copy("marketing", "onboardingInfoProcessBody")}
          />
          <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-6">
            {PROCESS_STEPS.map((step, index) => {
              const Icon = step.icon;
              return (
                <div key={step.title} className="relative rounded-2xl border border-slate-200 bg-background p-6">
                  <span className="absolute top-5 right-5 font-heading text-4xl font-bold text-slate-100">
                    {String(index + 1).padStart(2, "0")}
                  </span>
                  <div className="w-11 h-11 rounded-lg bg-emerald-500/10 text-emerald-600 flex items-center justify-center mb-4">
                    <Icon className="w-5 h-5" />
                  </div>
                  <h3 className="font-heading text-lg font-semibold text-slate-900 mb-2">{step.title}</h3>
                  <p className="text-sm text-slate-500 leading-relaxed">{step.body}</p>
                </div>
              );
            })}
          </div>
        </div>
      </section>

      {/* Checklist interactivo */}
      <section className="bg-background py-16 lg:py-24">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="grid lg:grid-cols-2 gap-12 items-center">
            <div>
              <SectionHeading
                align="left"
                eyebrow={copy("marketing", "onboardingInfoStepsLead")}
                title={copy("marketing", "onboardingInfoChecklistTitle")}
                lead={copy("marketing", "onboardingInfoChecklistBody")}
              />
            </div>
            <OnboardingInfoChecklist />
          </div>
        </div>
      </section>

      {/* Cronograma típico */}
      <section className="bg-white border-y border-slate-200 py-16 lg:py-24">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <SectionHeading title={copy("marketing", "onboardingInfoScheduleTitle")} />
          <div className="max-w-3xl mx-auto space-y-4">
            {SCHEDULE.map((item, index) => (
              <div
                key={item.day}
                className="flex items-start gap-4 rounded-xl border border-slate-200 bg-card p-5"
              >
                <span className="w-16 shrink-0 rounded-lg bg-emerald-500/10 text-emerald-700 text-sm font-bold px-2 py-1.5 text-center">
                  {item.day}
                </span>
                <p className="text-slate-700 font-medium leading-relaxed pt-1">{item.text}</p>
                {index < SCHEDULE.length - 1 && (
                  <span aria-hidden="true" className="hidden sm:block text-slate-300 ml-auto">
                    ↓
                  </span>
                )}
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* FAQ */}
      <section className="bg-background py-16 lg:py-24">
        <JsonLd id="onboarding-faq-jsonld" data={faqJsonLd} />
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <SectionHeading title={copy("marketing", "onboardingInfoFaqTitle")} />
          <div className="max-w-3xl mx-auto space-y-3">
            {faq.map((item) => (
              <details
                key={item.question}
                className="group rounded-xl border border-slate-200 bg-white px-5 py-4"
              >
                <summary className="flex items-center justify-between gap-4 cursor-pointer list-none text-slate-900 font-semibold">
                  {item.question}
                  <span className="text-emerald-600 text-lg shrink-0 transition-transform group-open:rotate-45">
                    +
                  </span>
                </summary>
                <p className="mt-3 text-sm text-slate-500 leading-relaxed">{item.answer}</p>
              </details>
            ))}
          </div>
        </div>
      </section>

      <CtaBanner />
    </>
  );
}
