import type { Metadata } from "next";
import { PageHero } from "@/components/marketing/page-hero";
import { copy } from "@/lib/copy/ui";

export function generateMetadata(): Metadata {
  return {
    title: copy("marketing", "pageTermsTitle"),
    description: copy("marketing", "pageLegalLead"),
  };
}

const CONTACT_EMAIL = "ventas@nexochat.co";

export default function TermsPage() {
  return (
    <>
      <PageHero
        eyebrow={copy("marketing", "navTerms")}
        title={copy("marketing", "pageTermsTitle")}
        lead={copy("marketing", "pageLegalLead")}
      />
      <section className="bg-background py-16 lg:py-24">
        <div className="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8">
          <p className="text-sm text-muted-foreground">
            Última actualización: 11 de agosto de 2026
          </p>
          <div className="mt-10 space-y-16">
            <div>
              <h2 className="font-heading text-2xl sm:text-3xl font-bold text-foreground tracking-tight">
                Uso del servicio
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Al crear una cuenta en NexoChat aceptas estos términos. El
                servicio permite gestionar conversaciones de WhatsApp e
                Instagram, entrenar el Copiloto de IA, generar facturación
                electrónica DIAN vía Siigo, enviar links de pago y ejecutar
                campañas, de acuerdo con las funcionalidades de tu plan.
              </p>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Te comprometes a usar el servicio de forma lícita, a no enviar
                mensajes no solicitados sin consentimiento y a cumplir la
                normativa de protección de datos aplicable, incluyendo la Ley
                1581 de Habeas Data en Colombia.
              </p>
            </div>

            <div>
              <h2 className="font-heading text-2xl sm:text-3xl font-bold text-foreground tracking-tight">
                Cuentas y responsabilidad
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Eres responsable de la información que registres en tu cuenta y
                de las acciones de los usuarios que la administren. El acceso
                es mediante autenticación sin contraseñas (enlaces de
                verificación por correo) y debes custodiar la bandeja de correo
                asociada, ya que con ella se autoriza el ingreso a la
                organización.
              </p>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Como responsable del tratamiento de los datos de tus clientes,
                garantizas contar con el consentimiento necesario para
                contactarlos y procesar su información a través de la
                plataforma.
              </p>
            </div>

            <div>
              <h2 className="font-heading text-2xl sm:text-3xl font-bold text-foreground tracking-tight">
                Pagos y facturación
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Los planes se facturan de forma mensual o anual según la opción
                elegida. Los pagos se procesan a través de MercadoPago (PSE,
                Nequi y tarjetas) y Polar (tarjetas internacionales). Al
                finalizar la compra se genera la facturación correspondiente a
                través de Siigo con resolución DIAN.
              </p>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Puedes cancelar tu suscripción en cualquier momento desde tu
                panel de facturación. No ofrecemos reembolsos proporcionales
                por periodos no consumidos, salvo disposición legal aplicable.
              </p>
            </div>

            <div>
              <h2 className="font-heading text-2xl sm:text-3xl font-bold text-foreground tracking-tight">
                Propiedad intelectual
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                La plataforma NexoChat, su código, diseño, marca y contenidos
                son propiedad de NexoChat o de sus licenciantes. Se te otorga
                una licencia limitada, no exclusiva e intransferible para usar
                el servicio durante la vigencia de tu plan. La información que
                tú cargues (precios, catálogos, historial de chats) sigue
                siendo tuya.
              </p>
            </div>

            <div>
              <h2 className="font-heading text-2xl sm:text-3xl font-bold text-foreground tracking-tight">
                Limitación de responsabilidad
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                El servicio se presta según su estado actual y disponibilidad.
                NexoChat no es responsable por interrupciones de proveedores
                externos (Meta, Siigo, MercadoPago, Polar), por decisiones
                comerciales tomadas con base en la plataforma ni por daños
                indirectos o lucro cesante derivados del uso del servicio.
                Nuestra responsabilidad agregada se limita al monto pagado por
                el plan en los últimos tres meses.
              </p>
            </div>

            <div>
              <h2 className="font-heading text-2xl sm:text-3xl font-bold text-foreground tracking-tight">
                Ley aplicable y jurisdicción
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Estos términos se rigen por las leyes de la República de
                Colombia. Cualquier controversia se someterá a la jurisdicción
                de los jueces de Bogotá D.C., salvo que la ley disponga un
                fuero de obligatorio cumplimiento.
              </p>
            </div>

            <div>
              <h2 className="font-heading text-2xl sm:text-3xl font-bold text-foreground tracking-tight">
                Contacto
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Para consultas sobre estos términos, escríbenos a{" "}
                <a
                  href={`mailto:${CONTACT_EMAIL}`}
                  className="text-primary font-medium underline underline-offset-2 hover:text-primary"
                >
                  {CONTACT_EMAIL}
                </a>
                .
              </p>
            </div>
          </div>
        </div>
      </section>
    </>
  );
}
