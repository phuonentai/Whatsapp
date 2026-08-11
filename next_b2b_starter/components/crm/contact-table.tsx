"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { Download, Upload } from "lucide-react";
import { useContactsQuery } from "@/lib/hooks/queries/use-crm-queries";
import { useDeleteContact } from "@/lib/hooks/mutations/use-crm-mutations";
import { useFeature } from "@/lib/hooks/use-entitlement";
import { usePermissions } from "@/lib/hooks/use-permissions";
import type { ContactDto } from "@/lib/api/api/dto/crm.dto";
import { crmRepository } from "@/lib/api/api/repositories/crm-repository";
import { ContactDialog } from "@/components/crm/contact-dialog";
import { ConfirmDialog } from "@/components/crm/confirm-dialog";
import { ContactImportDialog } from "@/components/crm/contact-import-dialog";
import { ErrorState } from "@/components/common/error-state";
import { Button } from "@/components/ui/button";

export function ContactTable() {
  const router = useRouter();
  const [search, setSearch] = useState("");
  const [filterStatus, setFilterStatus] = useState("");
  const [dialogOpen, setDialogOpen] = useState(false);
  const [importOpen, setImportOpen] = useState(false);
  const [editing, setEditing] = useState<ContactDto | null>(null);
  const [deleting, setDeleting] = useState<ContactDto | null>(null);
  const [isExporting, setIsExporting] = useState(false);
  const canManage = useFeature("crm_contacts_manage");
  const { hasPermission } = usePermissions();
  const canExport = hasPermission("contact:export");
  const { data: contacts, isLoading, isError, refetch, isRefetching } = useContactsQuery({ lead_status: filterStatus || undefined });
  const deleteMutation = useDeleteContact();

  const handleExport = async () => {
    setIsExporting(true);
    try {
      await crmRepository.exportContacts();
      toast.success("Contactos exportados");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Error al exportar contactos");
    } finally {
      setIsExporting(false);
    }
  };

  const filtered = useMemo(() => {
    const query = search.trim().toLowerCase();
    if (!query) return contacts;
    return contacts?.filter((c) =>
      [c.display_name, c.phone_number, c.email]
        .filter(Boolean)
        .some((value) => (value as string).toLowerCase().includes(query))
    );
  }, [contacts, search]);

  const handleDelete = async () => {
    if (!deleting) return;
    try {
      await deleteMutation.mutateAsync(deleting.id);
      toast.success("Contacto eliminado");
      setDeleting(null);
    } catch {
      // error toast handled by mutation
    }
  };

  if (isLoading) return <div className="text-gray-500">Cargando contactos...</div>;

  if (isError) {
    return (
      <ErrorState
        title="Error al cargar los contactos"
        description="No se pudieron cargar los contactos. Inténtalo de nuevo."
        onRetry={() => refetch()}
        isRetrying={isRefetching}
      />
    );
  }

  return (
    <div>
      <div className="flex gap-4 mb-4 items-center">
        <input
          type="text"
          placeholder="Buscar contactos..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="border rounded px-3 py-2 w-64"
        />
        <select
          value={filterStatus}
          onChange={(e) => setFilterStatus(e.target.value)}
          className="border rounded px-3 py-2"
        >
          <option value="">Todos los estados</option>
          <option value="nuevo">Nuevo</option>
          <option value="contactado">Contactado</option>
          <option value="calificado">Calificado</option>
          <option value="cliente">Cliente</option>
        </select>
        {canManage && (
          <Button
            onClick={() => {
              setEditing(null);
              setDialogOpen(true);
            }}
          >
            Nuevo contacto
          </Button>
        )}
        {canExport && (
          <Button variant="outline" onClick={handleExport} disabled={isExporting}>
            <Download className="mr-2 h-4 w-4" />
            {isExporting ? "Exportando..." : "Exportar"}
          </Button>
        )}
        {canManage && (
          <Button variant="outline" onClick={() => setImportOpen(true)}>
            <Upload className="mr-2 h-4 w-4" />
            Importar
          </Button>
        )}
      </div>

      <table className="w-full border-collapse">
        <thead>
          <tr className="bg-gray-100">
            <th className="text-left p-2">Nombre</th>
            <th className="text-left p-2">Teléfono</th>
            <th className="text-left p-2">Correo</th>
            <th className="text-left p-2">Documento</th>
            <th className="text-left p-2">Empresa</th>
            <th className="text-left p-2">Estado</th>
            <th className="text-left p-2">Último Contacto</th>
            <th className="text-left p-2">Acciones</th>
          </tr>
        </thead>
        <tbody>
          {filtered?.map((c) => (
            <tr
              key={c.id}
              className="border-b hover:bg-gray-50 cursor-pointer"
              onClick={() => router.push(`/dashboard/crm?view=contactos&id=${c.id}`)}
            >
              <td className="p-2 font-medium text-blue-600 hover:underline">{c.display_name || c.phone_number}</td>
              <td className="p-2">{c.phone_number}</td>
              <td className="p-2">{c.email || "-"}</td>
              <td className="p-2">{c.numero_documento ? `${c.tipo_documento} ${c.numero_documento}` : "-"}</td>
              <td className="p-2">{c.company_id ? `Empresa #${c.company_id}` : "-"}</td>
              <td className="p-2">
                <span className={`px-2 py-1 rounded text-xs ${
                  c.lead_status === "cliente" ? "bg-green-100 text-green-800" :
                  c.lead_status === "calificado" ? "bg-blue-100 text-blue-800" :
                  c.lead_status === "nuevo" ? "bg-gray-100 text-gray-800" :
                  "bg-yellow-100 text-yellow-800"
                }`}>
                  {c.lead_status}
                </span>
              </td>
              <td className="p-2 text-sm text-gray-500">
                {c.last_message_at ? new Date(c.last_message_at).toLocaleDateString("es-CO") : "-"}
              </td>
              <td className="p-2">
                {canManage && (
                  <div className="flex gap-2">
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        setEditing(c);
                        setDialogOpen(true);
                      }}
                      className="text-blue-600 hover:underline text-sm"
                    >
                      Editar
                    </button>
                    <button
                      aria-label="Eliminar"
                      onClick={(e) => {
                        e.stopPropagation();
                        setDeleting(c);
                      }}
                      className="text-red-600 hover:underline text-sm"
                    >
                      Eliminar
                    </button>
                  </div>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {(!filtered || filtered.length === 0) && (
        <div className="p-4 text-center text-gray-400">No hay contactos</div>
      )}

      <ContactDialog open={dialogOpen} onOpenChange={setDialogOpen} contact={editing} />
      <ContactImportDialog open={importOpen} onOpenChange={setImportOpen} />
      <ConfirmDialog
        open={Boolean(deleting)}
        onOpenChange={(next) => !next && setDeleting(null)}
        title="Eliminar contacto"
        description={`¿Estás seguro de eliminar a ${deleting?.display_name || deleting?.phone_number}? Esta acción no se puede deshacer.`}
        confirmLabel="Eliminar"
        loading={deleteMutation.isPending}
        onConfirm={handleDelete}
      />
    </div>
  );
}
