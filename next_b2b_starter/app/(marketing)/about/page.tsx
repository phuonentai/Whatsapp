import type { Metadata } from "next";
import {
  Check,
  Globe,
  ShieldCheck,
  Sparkles,
  Zap,
  type LucideIcon,
} from "lucide-react";
import { PageHero } from "@/components/marketing/page-hero";
import { CtaBanner } from "@/components/marketing/cta-banner";
import { Reveal } from "@/components/marketing/reveal";
import { copy } from "@/lib/copy/ui";

export function generateMetadata(): Metadata {
  return {
    title: copy("marketing", "pageAboutTitle"),
    description: copy("marketing", "pageAboutLead"),
  };
}

interface Value {
  icon: LucideIcon;
  title: string;
  description: string;
}

const VALUES: Value[] = [
  {
    icon: Zap,
    title: "Velocidad",
    description:
      "Respuestas en segundos, facturas con un clic y pagos dentro del chat. El tiempo de tu equipo se convierte en ventas.",
  },
  {
    icon: ShieldCheck,
    title: "Datos protegidos",
    description:
      "Cumplimos la Ley 1581 de Habeas Data: consentimiento, exportación y eliminación de datos personales siempre disponibles.",
  },
  {
    icon: Sparkles,
    title: "IA con control humano",
    description:
      "La IA propone respuestas y cotizaciones como borradores; tu equipo decide y aprueba antes de enviar o facturar.",
  },
  {
    icon: Globe,
    title: "Hecho para LATAM",
    description:
      "Pesos colombianos, PSE, Nequi, facturación DIAN con Siigo y WhatsApp: pensado para cómo se vende en la región.",
  },
];

const PRODUCT_PILLARS = [
  "Bandeja multi-canal de WhatsApp e Instagram",
  "Copiloto IA con base de conocimiento RAG",
  "Facturación electrónica Siigo DIAN",
  "Links de pago PSE, Nequi y tarjetas",
  "Campañas con consentimiento Ley 1581",
  "Analítica de ventas en COP",
];

export default function AboutPage() {
  return (
    <>
      <PageHero
        eyebrow="NexoChat"
        title={copy("marketing", "pageAboutTitle")}
        lead={copy("marketing", "pageAboutLead")}
      />

      {/* Misión */}
      <section className="bg-background py-16 lg:py-24">
        <div className="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8">
          <h2 className="font-heading text-3xl font-bold text-foreground tracking-tight">
            Nuestra misión
          </h2>
          <p className="mt-6 text-lg text-muted-foreground leading-relaxed">
            NexoChat existe para automatizar las ventas y la facturación por
            WhatsApp de las empresas en Colombia y LATAM. Miles de negocios
            todavía responden chats a mano, digitan cotizaciones dos veces y
            pierden ventas mientras el cliente espera.
          </p>
          <p className="mt-4 text-lg text-muted-foreground leading-relaxed">
            Unimos la API oficial de WhatsApp Business con un Copiloto de
            Inteligencia Artificial, facturación electrónica DIAN vía Siigo y
            cobros con PSE, Nequi y tarjetas, para que la conversación sea el
            canal donde se vende, se factura y se cobra.
          </p>
        </div>
      </section>

      {/* Valores */}
      <section className="bg-muted/50 py-16 lg:py-24">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <h2 className="font-heading text-3xl font-bold text-foreground tracking-tight">
            Nuestros valores
          </h2>
          <div className="mt-12 grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
            {VALUES.map((value) => {
              const Icon = value.icon;
              return (
                <div
                  key={value.title}
                  className="rounded-2xl border border-border bg-card p-6"
                >
                  <div className="w-11 h-11 rounded-lg bg-primary/10 text-primary flex items-center justify-center mb-4">
                    <Icon className="w-5 h-5" />
                  </div>
                  <h3 className="font-heading text-lg font-semibold text-foreground">
                    {value.title}
                  </h3>
                  <p className="mt-2 text-sm text-muted-foreground leading-relaxed">
                    {value.description}
                  </p>
                </div>
              );
            })}
          </div>
        </div>
      </section>

      {/* Producto */}
      <section className="bg-background py-16 lg:py-24">
        <div className="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8">
          <h2 className="font-heading text-3xl font-bold text-foreground tracking-tight">
            Qué es NexoChat
          </h2>
          <p className="mt-6 text-lg text-muted-foreground leading-relaxed">
            NexoChat es el CRM nativo para WhatsApp que automatiza ventas y
            facturación electrónica para empresas en Colombia y LATAM. Tu
            equipo atiende WhatsApp e Instagram en una sola bandeja, entrena la
            IA con sus propios documentos y cierra el ciclo de venta completo
            sin salir del chat.
          </p>
          <ul className="mt-8 grid gap-4 sm:grid-cols-2">
            {PRODUCT_PILLARS.map((pillar) => (
              <li
                key={pillar}
                className="flex items-start gap-3 rounded-xl border border-border bg-card p-5 text-sm text-muted-foreground leading-relaxed"
              >
                <Check className="w-5 h-5 mt-0.5 shrink-0 text-primary" />
                <span>{pillar}</span>
              </li>
            ))}
          </ul>
        </div>
      </section>

      <Reveal>
        <CtaBanner />
      </Reveal>
    </>
  );
}
