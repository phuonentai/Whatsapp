import type { Metadata } from "next";
import { PageHero } from "@/components/marketing/page-hero";
import { copy } from "@/lib/copy/ui";

export function generateMetadata(): Metadata {
  return {
    title: copy("marketing", "pagePrivacyTitle"),
    description: copy("marketing", "pageLegalLead"),
  };
}

const CONTACT_EMAIL = "ventas@nexochat.co";

export default function PrivacyPage() {
  return (
    <>
      <PageHero
        eyebrow={copy("marketing", "navPrivacy")}
        title={copy("marketing", "pagePrivacyTitle")}
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
                Datos personales que tratamos
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                NexoChat trata los datos que nos compartes al crear tu cuenta y
                usar el servicio: nombre, correo electrónico, organización y
                datos de facturación. Adicionalmente, cuando conectas tu
                WhatsApp o Instagram, procesamos los datos de tus clientes
                (nombres, números de teléfono y contenido de las
                conversaciones) para prestar el servicio de atención, venta y
                cobro dentro del chat.
              </p>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Nunca vendemos datos personales a terceros. Solo los
                compartimos con los proveedores estrictamente necesarios para
                operar el servicio (por ejemplo, Meta, Siigo, MercadoPago y
                Polar), bajo acuerdos de confidencialidad.
              </p>
            </div>

            <div>
              <h2 className="font-heading text-2xl sm:text-3xl font-bold text-foreground tracking-tight">
                Finalidad del tratamiento
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Tratamos los datos con finalidades específicas y legítimas:
                (i) prestar y administrar la plataforma, incluyendo la bandeja
                multi-canal, el Copiloto de IA y las integraciones; (ii)
                facturar el servicio y procesar pagos; (iii) cumplir
                obligaciones fiscales con la DIAN a través de Siigo; (iv)
                enviar comunicaciones sobre el servicio; y (v) mejorar la
                seguridad y el funcionamiento del producto.
              </p>
            </div>

            <div>
              <h2 className="font-heading text-2xl sm:text-3xl font-bold text-foreground tracking-tight">
                Consentimiento
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Cuando envías campañas de WhatsApp, NexoChat registra el
                consentimiento de cada contacto antes del primer envío y lo
                mantiene documentado. Los contactos pueden retirar su
                autorización en cualquier momento, y el sistema respeta esa
                decisión en los envíos posteriores.
              </p>
            </div>

            <div id="habeas-data">
              <h2 className="font-heading text-2xl sm:text-3xl font-bold text-foreground tracking-tight">
                Ley 1581 de Habeas Data
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                De acuerdo con la Ley 1581 de 2012 y sus decretos
                reglamentarios, como titular de datos personales tienes
                derecho a: conocer, actualizar y rectificar tus datos;
                solicitar prueba de la autorización otorgada; ser informado
                sobre el uso que se les ha dado; presentar quejas ante la
                Superintendencia de Industria y Comercio (SIC) cuando lo
                consideres necesario; y revocar la autorización o solicitar la
                supresión de los datos cuando ya no sean necesarios para la
                finalidad autorizada.
              </p>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Para ejercer cualquiera de estos derechos, escríbenos a{" "}
                <a
                  href={`mailto:${CONTACT_EMAIL}`}
                  className="text-primary font-medium underline underline-offset-2 hover:text-primary"
                >
                  {CONTACT_EMAIL}
                </a>{" "}
                indicando el derecho que deseas ejercer. Responderemos en los
                términos establecidos por la ley.
              </p>
            </div>

            <div>
              <h2 className="font-heading text-2xl sm:text-3xl font-bold text-foreground tracking-tight">
                Exportación y eliminación de datos
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Puedes exportar la información de tus conversaciones, contactos
                y facturas desde la plataforma en cualquier momento. Si
                solicitas la eliminación de tus datos, los eliminamos o
                anonimizamos de nuestros sistemas, excepto los que debamos
                conservar por obligaciones legales o fiscales, que retendremos
                únicamente durante el plazo exigido por la normativa
                colombiana.
              </p>
            </div>

            <div>
              <h2 className="font-heading text-2xl sm:text-3xl font-bold text-foreground tracking-tight">
                Retención de datos
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Conservamos los datos mientras tu cuenta esté activa y durante
                el tiempo necesario para cumplir obligaciones legales,
                contables y fiscales. Los datos de facturación electrónica se
                conservan conforme a los plazos exigidos por la DIAN. Al cerrar
                tu cuenta, los datos personales se eliminan o anonimizan salvo
                que exista una obligación legal de conservarlos.
              </p>
            </div>

            <div>
              <h2 className="font-heading text-2xl sm:text-3xl font-bold text-foreground tracking-tight">
                Cookies y tecnologías
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Usamos cookies y tecnologías similares para el funcionamiento
                de la plataforma (sesiones seguras y autenticación) y, en
                menor medida, para analítica agregada que nos ayuda a mejorar
                el producto. Puedes configurar tu navegador para rechazarlas,
                aunque algunas funciones del sitio podrían verse afectadas.
              </p>
            </div>

            <div>
              <h2 className="font-heading text-2xl sm:text-3xl font-bold text-foreground tracking-tight">
                Contacto
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Para consultas sobre esta política, el tratamiento de tus datos
                o el ejercicio de tus derechos de Habeas Data, escríbenos a{" "}
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
