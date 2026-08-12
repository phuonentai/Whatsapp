import type { Metadata } from "next";
import Link from "next/link";
import { ArrowRight, Bell, DollarSign, FileText, MessageCircle, Search, Zap } from "lucide-react";
import { Button } from "@/components/ui/button";
import { PageHero } from "@/components/marketing/page-hero";
import { SectionHeading } from "@/components/marketing/section-heading";
import { Reveal } from "@/components/marketing/reveal";
import { copy } from "@/lib/copy/ui";

export const metadata: Metadata = {
  title: "Plataforma",
  description:
    "Una sola herramienta para vender, facturar y atender por WhatsApp. Panel de control, bandeja de mensajes y Copiloto IA.",
};

const SIDEBAR_MAIN = [
  { label: "Panel", active: true },
  { label: "Conversaciones", badge: "24" },
  { label: "Contactos" },
  { label: "Facturas" },
  { label: "Pagos" },
  { label: "Analíticas" },
];

const SIDEBAR_AI = [
  { label: "Copiloto IA", badge: "Activo" },
  { label: "Entrenamiento" },
  { label: "Automatizaciones" },
];

const KPIS = [
  {
    value: "1,847",
    label: "Conversaciones activas",
    delta: "+23%",
    tone: "bg-primary/10 text-primary",
    icon: "chat",
  },
  {
    value: "$47.2M",
    label: "Ventas esta semana",
    delta: "+18%",
    tone: "bg-blue-100 text-blue-600",
    icon: "money",
  },
  {
    value: "284",
    label: "Facturas emitidas",
    delta: "+31%",
    tone: "bg-violet-100 text-violet-600",
    icon: "file",
  },
  {
    value: "1.2s",
    label: "Tiempo respuesta IA",
    delta: "+45%",
    tone: "bg-amber-100 text-amber-600",
    icon: "bolt",
  },
];

const CHART_DAYS = [32, 40, 28, 48, 36, 20, 16];

const INSIGHTS = [
  {
    tone: "bg-primary/10 text-primary",
    text: "Detecté 3 clientes con alta probabilidad de compra basado en su historial de conversación.",
    action: "Ver clientes →",
  },
  {
    tone: "bg-amber-500/10 text-amber-600",
    text: "8 conversaciones llevan más de 24h sin respuesta. Prioridad alta.",
    action: "Responder ahora →",
  },
  {
    tone: "bg-blue-500/10 text-blue-600",
    text: "Tus ventas aumentaron 32% vs semana pasada. Tendencia positiva.",
  },
];

const CONVERSATIONS = [
  {
    initials: "MG",
    name: "María González",
    tag: "Cliente",
    tagTone: "bg-primary/10 text-primary",
    snippet: "Pedido #4821: ¿me confirmas el envío?",
    time: "10:42 AM",
    unread: 2,
  },
  {
    initials: "CR",
    name: "Carlos Restrepo",
    tag: "Prospecto",
    tagTone: "bg-blue-100 text-blue-700",
    snippet: "Hola, quiero cotizar 50 unidades del producto A",
    time: "10:15 AM",
    unread: 0,
  },
  {
    initials: "AL",
    name: "Ana Lucía",
    tag: "Cliente",
    tagTone: "bg-primary/10 text-primary",
    snippet: "¡Gracias! Ya recibí la factura DIAN ✅",
    time: "9:58 AM",
    unread: 0,
  },
  {
    initials: "JP",
    name: "Jorge Pérez",
    tag: "Prospecto",
    tagTone: "bg-blue-100 text-blue-700",
    snippet: "¿Aceptan pagos con Nequi?",
    time: "9:31 AM",
    unread: 1,
  },
  {
    initials: "LS",
    name: "Laura Sánchez",
    tag: "Cliente",
    tagTone: "bg-primary/10 text-primary",
    snippet: "Me llegó el link de pago, listo ✅",
    time: "9:12 AM",
    unread: 0,
  },
];

function DashboardMock() {
  return (
    <div className="rounded-2xl border border-border bg-card shadow-xl shadow-slate-900/5 overflow-hidden">
      {/* Top bar */}
      <div className="bg-card border-b border-border px-4 py-3 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 bg-emerald-500 rounded-lg flex items-center justify-center">
            <svg className="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
            </svg>
          </div>
          <span className="font-heading font-bold text-lg text-foreground">NexoChat</span>
        </div>
        <div className="hidden md:flex items-center gap-4 text-muted-foreground">
          <Bell className="w-5 h-5" />
          <div className="w-8 h-8 bg-primary rounded-full flex items-center justify-center text-primary-foreground text-xs font-semibold">
            MR
          </div>
        </div>
      </div>
      <div className="grid md:grid-cols-[200px_1fr]">
        {/* Sidebar */}
        <aside className="hidden md:flex flex-col bg-muted/40 border-r border-border p-3">
          <nav className="space-y-1">
            {SIDEBAR_MAIN.map((item) => (
              <div
                key={item.label}
                className={`flex items-center gap-2 px-3 py-2 rounded-lg text-xs font-medium ${
                  item.active
                    ? "bg-primary/10 border border-primary/20 text-primary"
                    : "text-muted-foreground"
                }`}
              >
                {item.label}
                {item.badge && (
                  <span className="ml-auto bg-primary text-primary-foreground text-[10px] font-bold px-1.5 py-0.5 rounded-full">
                    {item.badge}
                  </span>
                )}
              </div>
            ))}
          </nav>
          <p className="text-[10px] font-semibold text-muted-foreground uppercase tracking-wider mt-4 mb-1 px-3">
            Inteligencia Artificial
          </p>
          <nav className="space-y-1">
            {SIDEBAR_AI.map((item) => (
              <div key={item.label} className="flex items-center gap-2 px-3 py-2 rounded-lg text-xs font-medium text-muted-foreground">
                {item.label}
                {item.badge && (
                  <span className="ml-auto text-[10px] bg-primary/10 text-primary px-1.5 py-0.5 rounded-full">
                    {item.badge}
                  </span>
                )}
              </div>
            ))}
          </nav>
        </aside>
        {/* Main */}
        <div className="bg-muted/30 p-5">
          <div className="flex items-center justify-between mb-4">
            <div>
              <p className="font-heading text-lg font-bold text-foreground">
                {copy("marketing", "plataformaMockGreeting")}, María 👋
              </p>
              <p className="text-xs text-muted-foreground">Miércoles, 15 de Enero 2026</p>
            </div>
            <Button className="hidden sm:inline-flex bg-primary hover:bg-primary/90 text-primary-foreground text-xs py-2">
              {copy("marketing", "plataformaMockNewCampaign")}
            </Button>
          </div>
          <div className="grid grid-cols-2 xl:grid-cols-4 gap-3 mb-4">
            {KPIS.map((kpi) => (
              <div key={kpi.label} className="bg-card rounded-xl p-4 border border-border">
                <div className="flex items-start justify-between mb-2">
                  <div className={`w-8 h-8 rounded-lg flex items-center justify-center ${kpi.tone}`}>
                    {kpi.icon === "chat" && <MessageCircle className="w-4 h-4" />}
                    {kpi.icon === "money" && <DollarSign className="w-4 h-4" />}
                    {kpi.icon === "file" && <FileText className="w-4 h-4" />}
                    {kpi.icon === "bolt" && <Zap className="w-4 h-4" />}
                  </div>
                  <span className="text-[10px] font-medium text-primary bg-primary/10 px-1.5 py-0.5 rounded">
                    {kpi.delta}
                  </span>
                </div>
                <div className="text-xl font-bold text-foreground">{kpi.value}</div>
                <div className="text-[11px] text-muted-foreground">{kpi.label}</div>
              </div>
            ))}
          </div>
          <div className="grid xl:grid-cols-3 gap-4">
            <div className="xl:col-span-2 bg-card rounded-xl border border-border p-4">
              <div className="flex items-center justify-between mb-3">
                <h3 className="font-heading text-sm font-bold text-foreground">Rendimiento de Ventas</h3>
                <span className="text-[10px] text-muted-foreground">Comparativa mensual con predicción IA</span>
              </div>
              <div className="h-28 flex items-end justify-between gap-2">
                {CHART_DAYS.map((h, i) => (
                  <div key={i} className="flex-1 flex flex-col items-center gap-1">
                    <div className="w-full flex flex-col gap-1 justify-end" style={{ height: 100 }}>
                      <div className="w-full bg-primary/30 rounded-t-lg" style={{ height: h * 0.3 }} />
                      <div className="w-full bg-primary rounded-t-lg" style={{ height: h * 0.7 }} />
                    </div>
                  </div>
                ))}
              </div>
            </div>
            <div className="bg-primary rounded-xl p-4 text-primary-foreground">
              <div className="flex items-center gap-2 mb-3">
                <span className="text-sm font-bold">{copy("marketing", "plataformaMockCopilotTitle")}</span>
                <span className="text-[10px] text-primary-foreground/70">· {copy("marketing", "plataformaMockCopilotSub")}</span>
              </div>
              <div className="space-y-2">
                {INSIGHTS.map((ins, i) => (
                  <div key={i} className="bg-primary-foreground/10 rounded-lg p-3 border border-primary-foreground/20">
                    <p className="text-[11px] text-primary-foreground/90 leading-relaxed">{ins.text}</p>
                    {ins.action && (
                      <button className="text-[10px] text-primary-foreground font-medium mt-1 hover:underline">
                        {ins.action}
                      </button>
                    )}
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function MessagesMock() {
  return (
    <div className="rounded-2xl border border-border bg-card shadow-xl shadow-slate-900/5 overflow-hidden">
      <div className="p-5 border-b border-border flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div>
          <h3 className="font-heading text-lg font-bold text-foreground">Mensajes</h3>
          <p className="text-xs text-muted-foreground">Gestiona tus conversaciones de WhatsApp en un solo lugar</p>
        </div>
        <div className="flex items-center gap-2">
          <span className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-background border border-border rounded-lg text-xs text-muted-foreground font-medium">
            <span className="w-1.5 h-1.5 bg-primary rounded-full" />
            {copy("marketing", "plataformaMockConnected")}
          </span>
          <Button className="bg-primary hover:bg-primary/90 text-primary-foreground text-xs">
            {copy("marketing", "plataformaMockNewCampaign")}
          </Button>
        </div>
      </div>
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 p-5 border-b border-border">
        <div className="bg-card rounded-xl p-4 border border-border">
          <div className="text-2xl font-bold text-foreground">247</div>
          <div className="text-xs text-muted-foreground">{copy("marketing", "plataformaMockConversationsToday")}</div>
          <div className="text-[11px] text-primary mt-1">↑ 12% vs ayer</div>
        </div>
        <div className="bg-card rounded-xl p-4 border border-border">
          <div className="text-2xl font-bold text-foreground">18</div>
          <div className="text-xs text-muted-foreground">{copy("marketing", "plataformaMockPending")}</div>
          <div className="text-[11px] text-amber-600 mt-1">⚠ 5 urgentes</div>
        </div>
        <div className="bg-card rounded-xl p-4 border border-border">
          <div className="text-2xl font-bold text-foreground">94%</div>
          <div className="text-xs text-muted-foreground">{copy("marketing", "plataformaMockResponseRate")}</div>
          <div className="text-[11px] text-primary mt-1">↑ 3% este mes</div>
        </div>
        <div className="bg-card rounded-xl p-4 border border-border">
          <div className="text-2xl font-bold text-foreground">4.2m</div>
          <div className="text-xs text-muted-foreground">{copy("marketing", "plataformaMockAvgTime")}</div>
          <div className="text-[11px] text-primary mt-1">↓ Mejorando</div>
        </div>
      </div>
      <div className="p-4 border-b border-border flex flex-col sm:flex-row gap-3">
        <div className="relative flex-1 max-w-md">
          <Search className="w-4 h-4 text-muted-foreground absolute left-3 top-1/2 -translate-y-1/2" />
          <input
            type="text"
            placeholder={copy("marketing", "plataformaMockSearch")}
            className="w-full pl-9 pr-3 py-2 bg-muted border border-border rounded-lg text-xs focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
            aria-label={copy("marketing", "plataformaMockSearch")}
          />
        </div>
        <div className="flex items-center gap-2">
          <span className="px-3 py-2 bg-muted border border-border rounded-lg text-xs text-muted-foreground">
            {copy("marketing", "plataformaMockAllStates")}
          </span>
          <span className="px-3 py-2 bg-muted border border-border rounded-lg text-xs text-muted-foreground">
            {copy("marketing", "plataformaMockAllAgents")}
          </span>
        </div>
      </div>
      <div className="divide-y divide-border">
        {CONVERSATIONS.map((c) => (
          <div key={c.name} className="flex items-start gap-3 p-4 hover:bg-muted/50 transition-colors">
            <div className="relative flex-shrink-0">
              <div className="w-10 h-10 rounded-full bg-primary/10 text-primary flex items-center justify-center text-xs font-bold">
                {c.initials}
              </div>
              <span className="absolute -bottom-0.5 -right-0.5 w-3 h-3 bg-primary border-2 border-card rounded-full" />
            </div>
            <div className="flex-1 min-w-0">
              <div className="flex items-center justify-between gap-2">
                <h4 className="text-sm font-semibold text-foreground truncate">{c.name}</h4>
                <span className="text-[11px] text-muted-foreground flex-shrink-0">{c.time}</span>
              </div>
              <div className="flex items-center gap-2 mt-0.5">
                <span className={`inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium ${c.tagTone}`}>
                  {c.tag}
                </span>
                <span className="text-xs text-muted-foreground truncate">{c.snippet}</span>
              </div>
            </div>
            {c.unread > 0 && (
              <span className="w-5 h-5 bg-primary text-primary-foreground text-[10px] font-bold rounded-full flex items-center justify-center flex-shrink-0">
                {c.unread}
              </span>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

export default function PlataformaPage() {
  return (
    <>
      <PageHero
        eyebrow={copy("marketing", "navPlataforma")}
        title={copy("marketing", "pagePlataformaTitle")}
        lead={copy("marketing", "pagePlataformaLead")}
      />
      {/* Dashboard overview mock */}
      <section className="py-16 lg:py-24 bg-muted/50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <SectionHeading
            eyebrow={copy("marketing", "navPlataforma")}
            title={copy("marketing", "plataformaOverviewTitle")}
            lead={copy("marketing", "plataformaOverviewLead")}
          />
          <Reveal>
            <DashboardMock />
          </Reveal>
        </div>
      </section>
      {/* Messages view mock */}
      <section className="py-16 lg:py-24 bg-background">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <SectionHeading
            eyebrow={copy("marketing", "navPlataforma")}
            title={copy("marketing", "plataformaMessagesTitle")}
            lead={copy("marketing", "plataformaMessagesLead")}
          />
          <Reveal delay={0.1}>
            <MessagesMock />
          </Reveal>
        </div>
      </section>
      {/* CTA */}
      <section className="py-16 lg:py-24 bg-card border-t border-border">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
          <h2 className="font-heading text-3xl sm:text-4xl font-bold text-foreground mb-4">
            {copy("marketing", "plataformaCtaTitle")}
          </h2>
          <p className="text-lg text-muted-foreground mb-8 max-w-2xl mx-auto">
            {copy("marketing", "plataformaCtaLead")}
          </p>
          <Link href="/signup">
            <Button className="bg-primary hover:bg-primary/90 text-primary-foreground px-8 py-4 rounded-xl font-bold text-lg">
              {copy("marketing", "plataformaCtaButton")}
              <ArrowRight className="w-5 h-5" />
            </Button>
          </Link>
        </div>
      </section>
    </>
  );
}
