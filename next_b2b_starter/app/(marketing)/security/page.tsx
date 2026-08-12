import type { Metadata } from "next";
import { PageHero } from "@/components/marketing/page-hero";
import { copy } from "@/lib/copy/ui";

export function generateMetadata(): Metadata {
  return {
    title: copy("marketing", "pageSecurityTitle"),
    description: copy("marketing", "pageLegalLead"),
  };
}

export default function SecurityPage() {
  return (
    <>
      <PageHero
        eyebrow={copy("marketing", "navSecurity")}
        title={copy("marketing", "pageSecurityTitle")}
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
                Autenticación sin contraseñas y RBAC
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                NexoChat usa <strong className="font-semibold text-foreground">Stytch B2B</strong>{" "}
                para la autenticación de los equipos: acceso sin contraseñas
                mediante enlaces de verificación por correo, lo que elimina el
                riesgo de credenciales débiles o reutilizadas. El acceso está
                organizado por organización y cada miembro solo puede ingresar
                con el correo autorizado por su administrador.
              </p>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Sobre esa base aplicamos <strong className="font-semibold text-foreground">control de acceso por
                roles (RBAC)</strong> con permisos finos: quién puede ver
                conversaciones, administrar la facturación, entrenar la IA o
                gestionar campañas depende del rol asignado, no del criterio de
                cada usuario.
              </p>
            </div>

            <div>
              <h2 className="font-heading text-2xl sm:text-3xl font-bold text-foreground tracking-tight">
                Encriptación de datos
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Los datos almacenados en NexoChat se cifran{" "}
                <strong className="font-semibold text-foreground">en reposo</strong> con
                cifrado AES-256 a nivel de bases de datos y de copias de
                respaldo. Las credenciales de integración (Meta WhatsApp, Siigo,
                MercadoPago, Polar) se guardan cifradas y nunca se exponen en
                texto plano.
              </p>
            </div>

            <div>
              <h2 className="font-heading text-2xl sm:text-3xl font-bold text-foreground tracking-tight">
                Cifrado en tránsito
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Todo el tráfico entre tu equipo y NexoChat viaja cifrado con{" "}
                <strong className="font-semibold text-foreground">TLS 1.2 o superior</strong>{" "}
                (1.3 cuando el cliente lo soporta). La política de seguridad de
                transporte (HSTS) está habilitada para que los navegadores y
                aplicaciones siempre usen conexiones cifradas, y nuestras
                integraciones con WhatsApp, Siigo y los procesadores de pago se
                realizan exclusivamente por canales autenticados y cifrados.
              </p>
            </div>

            <div>
              <h2 className="font-heading text-2xl sm:text-3xl font-bold text-foreground tracking-tight">
                Resiliencia y circuit breakers
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                NexoChat depende de proveedores externos como Meta, Siigo,
                MercadoPago y Polar. Para aislar fallos de cualquiera de ellos
                usamos <strong className="font-semibold text-foreground">circuit
                breakers</strong>, timeouts y reintentos con retroceso
                exponencial: si un proveedor se degrada, la conversación sigue
                funcionando y el error se aísla sin afectar al resto del
                servicio.
              </p>
            </div>

            <div>
              <h2 className="font-heading text-2xl sm:text-3xl font-bold text-foreground tracking-tight">
                Cumplimiento Ley 1581 de Habeas Data
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Nuestro producto incorpora los mecanismos que exige la Ley 1581
                de 2012 y sus decretos reglamentarios en Colombia: máquina de
                consentimiento para campañas, registro del consentimiento por
                contacto y herramientas de exportación y eliminación de datos
                personales. Consulta la{" "}
                <a
                  href="/privacy#habeas-data"
                  className="text-primary font-medium underline underline-offset-2 hover:text-primary"
                >
                  política de privacidad
                </a>{" "}
                para conocer los derechos que tienes sobre tus datos.
              </p>
            </div>

            <div>
              <h2 className="font-heading text-2xl sm:text-3xl font-bold text-foreground tracking-tight">
                Backups y continuidad
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Realizamos <strong className="font-semibold text-foreground">copias de
                seguridad cifradas y periódicas</strong> de la información, con
                verificación de restauración y un plan de recuperación ante
                desastres para restablecer el servicio en el menor tiempo
                posible.
              </p>
            </div>
          </div>
        </div>
      </section>
    </>
  );
}
