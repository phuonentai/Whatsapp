"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { useContactQuery, useContactActivitiesQuery, useDealsQuery } from "@/lib/hooks/queries/use-crm-queries";
import { useCreateActivity, useUpdateContact } from "@/lib/hooks/mutations/use-crm-mutations";
import { useFeature } from "@/lib/hooks/use-entitlement";
import { ContactDialog } from "@/components/crm/contact-dialog";
import { TagPicker } from "@/components/crm/tag-picker";
import { ErrorState } from "@/components/common/error-state";
import { Button } from "@/components/ui/button";

export function ContactDetail({ id }: { id: number }) {
  const router = useRouter();
  const { data: contact, isLoading, isError, refetch, isRefetching } = useContactQuery(id);
  const { data: negocios } = useDealsQuery({ contact_id: id });
  const { data: activities } = useContactActivitiesQuery(id);
  const createActivity = useCreateActivity();
  const canManage = useFeature("crm_contacts_manage");
  const [dialogOpen, setDialogOpen] = useState(false);
  const [noting, setNoting] = useState(false);
  const [notaAsunto, setNotaAsunto] = useState("");
  const [notaContenido, setNotaContenido] = useState("");

  if (isLoading) return <div className="text-gray-500">Cargando contacto...</div>;

  if (isError) {
    return (
      <ErrorState
        title="Error al cargar el contacto"
        description="No se pudo cargar el contacto. Inténtalo de nuevo."
        onRetry={() => refetch()}
        isRetrying={isRefetching}
      />
    );
  }

  if (!contact) return <div className="text-gray-500">Contacto no encontrado</div>;

  const handleAddNote = () => {
    createActivity.mutate(
      { contact_id: id, tipo: "nota", asunto: notaAsunto, contenido: notaContenido } as never,
      {
        onSuccess: () => {
          toast.success("Nota agregada");
          setNoting(false);
          setNotaAsunto("");
          setNotaContenido("");
        },
      }
    );
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <button
          onClick={() => router.back()}
          className="text-blue-600 hover:underline text-sm"
        >
          ← Volver
        </button>
        <div className="flex gap-2">
          {canManage && (
            <>
              <Button variant="outline" onClick={() => setNoting(!noting)}>
                Agregar nota
              </Button>
              <Button variant="outline" onClick={() => router.push("/dashboard/crm?view=negocios")}>
                Crear negocio
              </Button>
              <Button
                onClick={() => {
                  setDialogOpen(true);
                }}
              >
                Editar
              </Button>
            </>
          )}
        </div>
      </div>

      <div className="bg-white rounded-lg border p-4 mb-4">
        <h2 className="text-xl font-semibold mb-3">
          {contact.display_name || contact.phone_number}
        </h2>
        <dl className="grid grid-cols-2 gap-3 text-sm">
          <div><dt className="text-gray-500">Teléfono</dt><dd>{contact.phone_number}</dd></div>
          <div><dt className="text-gray-500">Correo</dt><dd>{contact.email || "-"}</dd></div>
          <div><dt className="text-gray-500">Tipo Documento</dt><dd>{contact.tipo_documento || "-"}</dd></div>
          <div><dt className="text-gray-500">Número Documento</dt><dd>{contact.numero_documento || "-"}</dd></div>
          <div><dt className="text-gray-500">Empresa</dt><dd>{contact.company_id ? `Empresa #${contact.company_id}` : "-"}</dd></div>
          <div><dt className="text-gray-500">Estado</dt><dd>{contact.lead_status}</dd></div>
        </dl>
        <TagPicker entityType="contact" entityId={contact.id} />
      </div>

      <div className="bg-white rounded-lg border p-4 mb-4">
        <h3 className="font-semibold mb-2">Negocios asociados</h3>
        {negocios && negocios.length > 0 ? (
          <ul className="divide-y">
            {negocios.map((d) => (
              <li key={d.id} className="py-2 text-sm flex justify-between">
                <span>{d.nombre}</span>
                <span className="text-gray-500">{d.estado}</span>
              </li>
            ))}
          </ul>
        ) : (
          <p className="text-sm text-gray-400">Sin negocios asociados</p>
        )}
      </div>

      <div className="bg-white rounded-lg border p-4">
        <h3 className="font-semibold mb-2">Actividad</h3>
        {noting && (
          <div className="bg-gray-50 rounded p-3 mb-3 border">
            <input
              type="text"
              placeholder="Asunto"
              value={notaAsunto}
              onChange={(e) => setNotaAsunto(e.target.value)}
              className="border rounded px-3 py-2 w-full mb-2"
            />
            <textarea
              placeholder="Contenido"
              value={notaContenido}
              onChange={(e) => setNotaContenido(e.target.value)}
              className="border rounded px-3 py-2 w-full mb-2"
              rows={2}
            />
            <Button onClick={handleAddNote} disabled={createActivity.isPending}>
              Guardar nota
            </Button>
          </div>
        )}
        <div className="space-y-2">
          {activities?.map((a) => (
            <div key={a.id} className="text-sm border-b pb-2">
              <div className="font-medium">{a.asunto || a.tipo}</div>
              {a.contenido && <div className="text-gray-600">{a.contenido}</div>}
              <div className="text-xs text-gray-400">
                {new Date(a.realizada_en).toLocaleDateString("es-CO", {
                  day: "numeric", month: "long", hour: "2-digit", minute: "2-digit",
                })}
                {a.realizada_por_nombre && ` • ${a.realizada_por_nombre}`}
              </div>
            </div>
          ))}
          {(!activities || activities.length === 0) && (
            <p className="text-sm text-gray-400">Sin actividad registrada</p>
          )}
        </div>
      </div>

      <ContactDialog open={dialogOpen} onOpenChange={setDialogOpen} contact={contact} />
    </div>
  );
}
