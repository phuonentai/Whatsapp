"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { Download } from "lucide-react";
import { useCompaniesQuery } from "@/lib/hooks/queries/use-crm-queries";
import { useDeleteCompany } from "@/lib/hooks/mutations/use-crm-mutations";
import { useFeature } from "@/lib/hooks/use-entitlement";
import { usePermissions } from "@/lib/hooks/use-permissions";
import type { CompanyDto } from "@/lib/api/api/dto/crm.dto";
import { crmRepository } from "@/lib/api/api/repositories/crm-repository";
import { CompanyDialog } from "@/components/crm/company-dialog";
import { ConfirmDialog } from "@/components/crm/confirm-dialog";
import { Button } from "@/components/ui/button";

export function CompanyTable() {
  const router = useRouter();
  const [search, setSearch] = useState("");
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<CompanyDto | null>(null);
  const [deleting, setDeleting] = useState<CompanyDto | null>(null);
  const [isExporting, setIsExporting] = useState(false);
  const hasCompanies = useFeature("crm_companies");
  const { hasPermission } = usePermissions();
  const canExport = hasPermission("contact:export");
  const { data: companies, isLoading } = useCompaniesQuery();
  const deleteMutation = useDeleteCompany();

  const handleExport = async () => {
    setIsExporting(true);
    try {
      await crmRepository.exportCompanies();
      toast.success("Empresas exportadas");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Error al exportar empresas");
    } finally {
      setIsExporting(false);
    }
  };

  const filtered = useMemo(() => {
    const query = search.trim().toLowerCase();
    if (!query) return companies;
    return companies?.filter((c) =>
      [c.name, c.nit, c.sector].filter(Boolean).some((value) => (value as string).toLowerCase().includes(query))
    );
  }, [companies, search]);

  const handleDelete = async () => {
    if (!deleting) return;
    await deleteMutation.mutateAsync(deleting.id);
    toast.success("Empresa eliminada");
    setDeleting(null);
  };

  if (isLoading) return <div className="text-gray-500">Cargando empresas...</div>;

  return (
    <div>
      <div className="flex gap-4 mb-4 items-center">
        <input
          type="text"
          placeholder="Buscar empresas por nombre, NIT o sector..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="border rounded px-3 py-2 w-64"
        />
        {hasCompanies && (
          <Button
            onClick={() => {
              setEditing(null);
              setDialogOpen(true);
            }}
          >
            Nueva empresa
          </Button>
        )}
        {hasCompanies && canExport && (
          <Button variant="outline" onClick={handleExport} disabled={isExporting}>
            <Download className="mr-2 h-4 w-4" />
            {isExporting ? "Exportando..." : "Exportar"}
          </Button>
        )}
      </div>

      <table className="w-full border-collapse">
        <thead>
          <tr className="bg-gray-100">
            <th className="text-left p-2">Nombre</th>
            <th className="text-left p-2">NIT</th>
            <th className="text-left p-2">Sector</th>
            <th className="text-left p-2">Ciudad</th>
            <th className="text-left p-2">Tipo</th>
            <th className="text-left p-2">Contactos</th>
            <th className="text-left p-2">Acciones</th>
          </tr>
        </thead>
        <tbody>
          {filtered?.map((c) => (
            <tr
              key={c.id}
              className="border-b hover:bg-gray-50 cursor-pointer"
              onClick={() => router.push(`/dashboard/crm?view=empresas&id=${c.id}`)}
            >
              <td className="p-2 font-medium text-blue-600 hover:underline">{c.name}</td>
              <td className="p-2">{c.nit || "-"}</td>
              <td className="p-2">{c.sector || "-"}</td>
              <td className="p-2">{c.ciudad || "-"}</td>
              <td className="p-2">{c.tipo_empresa || "-"}</td>
              <td className="p-2">{c.total_contactos || 0}</td>
              <td className="p-2">
                {hasCompanies && (
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
        <div className="p-4 text-center text-gray-400">No hay empresas</div>
      )}

      <CompanyDialog open={dialogOpen} onOpenChange={setDialogOpen} company={editing} />
      <ConfirmDialog
        open={Boolean(deleting)}
        onOpenChange={(next) => !next && setDeleting(null)}
        title="Eliminar empresa"
        description={`¿Estás seguro de eliminar la empresa ${deleting?.name}? Esta acción no se puede deshacer.`}
        confirmLabel="Eliminar"
        loading={deleteMutation.isPending}
        onConfirm={handleDelete}
      />
    </div>
  );
}
