import Link from "next/link";
import { ArrowRight, FileText, Zap } from "lucide-react";
import { Button } from "@/components/ui/button";
import { copy } from "@/lib/copy/ui";

const INTEGRATIONS = ["Meta WhatsApp", "Siigo", "MercadoPago", "Nequi", "Efecty"];

export function Hero() {
  return (
    <section className="relative bg-slate-900 text-white overflow-hidden">
      {/* Sutiles acentos emerald de la identidad Verifika (sin gradientes pesados) */}
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_at_top_right,rgba(16,185,129,0.18),transparent_55%)]"
      />
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_at_bottom_left,rgba(16,185,129,0.10),transparent_45%)]"
      />
      <div className="relative max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16 lg:py-24">
        <div className="grid lg:grid-cols-12 gap-12 lg:gap-16 items-center">
          {/* Copy column */}
          <div className="lg:col-span-6 space-y-8">
            <div className="inline-flex items-center gap-2 bg-emerald-500/10 border border-emerald-500/30 rounded-full px-4 py-2">
              <span className="w-2 h-2 bg-emerald-400 rounded-full" />
              <span className="text-emerald-400 text-sm font-medium">
                {copy("marketing", "heroBadge")}
              </span>
            </div>
            <h1 className="font-heading text-4xl sm:text-5xl font-bold leading-tight tracking-tight text-balance">
              {copy("marketing", "heroTitle")}{" "}
              <span className="text-emerald-400">
                {copy("marketing", "heroTitleAccent")}
              </span>{" "}
              {copy("marketing", "heroTitleSuffix")}
            </h1>
            <p className="text-lg text-slate-400 leading-relaxed max-w-2xl">
              {copy("marketing", "heroLead")}
            </p>
            <div className="flex flex-col sm:flex-row gap-4">
              <Link href="/signup">
                <Button
                  size="lg"
                  className="bg-emerald-500 hover:bg-emerald-600 text-white rounded-xl font-semibold"
                >
                  {copy("marketing", "heroCtaPrimary")}
                  <ArrowRight className="w-5 h-5" />
                </Button>
              </Link>
              <Link href="/features">
                <Button
                  size="lg"
                  variant="outline"
                  className="border-slate-700 text-white rounded-xl font-semibold hover:bg-slate-800 hover:border-slate-600"
                >
                  {copy("marketing", "heroCtaSecondary")}
                </Button>
              </Link>
            </div>
            {/* Integrations strip */}
            <div className="border-t border-slate-800 pt-8">
              <p className="text-slate-400 text-sm mb-4">
                {copy("marketing", "integrationsLabel")}
              </p>
              <div className="flex flex-wrap gap-x-8 gap-y-3 text-slate-300 font-semibold text-sm">
                {INTEGRATIONS.map((name) => (
                  <span key={name}>{name}</span>
                ))}
              </div>
            </div>
          </div>

          {/* Chat mockup column */}
          <div className="lg:col-span-6 relative lg:pl-6">
            <div className="relative rounded-2xl bg-slate-800/80 border border-slate-700 shadow-xl shadow-black/30 overflow-hidden">
              {/* macOS-style header */}
              <div className="bg-slate-800 px-4 py-3 border-b border-slate-700 flex items-center gap-2">
                <span className="w-3 h-3 rounded-full bg-red-400" />
                <span className="w-3 h-3 rounded-full bg-yellow-400" />
                <span className="w-3 h-3 rounded-full bg-slate-600" />
              </div>
              <div className="grid grid-cols-2 divide-x divide-slate-700">
                {/* WhatsApp chat panel */}
                <div className="p-4 bg-slate-800/60">
                  <div className="text-xs text-slate-400 mb-3 uppercase tracking-wider">
                    {copy("marketing", "heroMockChatLabel")}
                  </div>
                  <div className="space-y-3">
                    <div className="bg-slate-700 rounded-lg p-3 text-sm text-white max-w-[80%]">
                      Hola, quiero cotizar 50 unidades del producto A
                    </div>
                    <div className="bg-emerald-600 rounded-lg p-3 text-sm text-white max-w-[80%] ml-auto">
                      ¡Claro! Generando cotización...
                    </div>
                    <div className="bg-emerald-600 rounded-lg p-3 text-sm text-white max-w-[80%] ml-auto flex items-center gap-2">
                      <FileText className="w-4 h-4" />
                      Factura #001 generada
                    </div>
                  </div>
                </div>
                {/* Siigo invoice panel */}
                <div className="p-4 bg-slate-800">
                  <div className="text-xs text-slate-400 mb-3 uppercase tracking-wider">
                    {copy("marketing", "heroMockInvoiceLabel")}
                  </div>
                  <div className="bg-slate-700/50 rounded-lg p-4 border border-slate-700">
                    <div className="flex items-center justify-between mb-4">
                      <span className="text-emerald-400 text-sm font-semibold">
                        Factura Electrónica
                      </span>
                      <span className="bg-emerald-500/15 text-emerald-400 text-xs px-2 py-1 rounded">
                        {copy("marketing", "heroMockInvoiceBadge")}
                      </span>
                    </div>
                    <div className="space-y-2 text-sm">
                      <div className="flex justify-between text-slate-400">
                        <span>Producto A x50</span>
                        <span>$2.500.000</span>
                      </div>
                      <div className="flex justify-between text-slate-400">
                        <span>IVA (19%)</span>
                        <span>$475.000</span>
                      </div>
                      <div className="border-t border-slate-700 pt-2 flex justify-between text-white font-bold">
                        <span>Total</span>
                        <span>$2.975.000</span>
                      </div>
                    </div>
                    <button
                      type="button"
                      className="w-full mt-4 bg-emerald-500 hover:bg-emerald-600 text-white py-2 rounded-lg text-sm font-semibold"
                    >
                      Enviar Link de Pago
                    </button>
                  </div>
                </div>
              </div>
            </div>
            {/* Floating stat card */}
            <div className="absolute -bottom-6 -left-6 bg-slate-800 rounded-xl shadow-lg shadow-black/30 border border-slate-700 p-4 hidden lg:flex items-center gap-3">
              <div className="w-10 h-10 rounded-lg bg-emerald-500/10 flex items-center justify-center text-emerald-400">
                <Zap className="w-5 h-5" />
              </div>
              <div>
                <p className="text-2xl font-bold text-white leading-none">
                  {copy("marketing", "heroStatValue")}
                </p>
                <p className="text-xs text-slate-400 mt-1">
                  {copy("marketing", "heroStatLabel")}
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
