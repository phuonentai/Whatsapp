import type { Metadata } from "next";
import { SiteHeader } from "@/components/marketing/site-header";
import { SiteFooter } from "@/components/marketing/site-footer";

export const metadata: Metadata = {
  title: {
    default: "NexoChat | CRM nativo para WhatsApp con IA",
    template: "%s | NexoChat",
  },
  description:
    "El CRM nativo para WhatsApp que automatiza ventas, facturación electrónica DIAN con Siigo y cobros PSE/Nequi para empresas en Colombia y LATAM.",
};

export default function MarketingLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <>
      <SiteHeader />
      <main>{children}</main>
      <SiteFooter />
    </>
  );
}
