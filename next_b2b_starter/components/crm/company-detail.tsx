"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useCompanyQuery, useDealsQuery } from "@/lib/hooks/queries/use-crm-queries";
import { useFeature } from "@/lib/hooks/use-entitlement";
import { CompanyDialog } from "@/components/crm/company-dialog";
import { TagPicker } from "@/components/crm/tag-picker";
import { Button } from "@/components/ui/button";

export function CompanyDetail({ id }: { id: number }) {
  const router = useRouter();
  const { data: company, isLoading } = useCompanyQuery(id);
  const { data: negocios } = useDealsQuery();
  const canManage = useFeature("crm_companies");
  const [dialogOpen, setDialogOpen] = useState(false);

  if (isLoading) return <div className="text-gray-500">Cargando empresa...</div>;
  if (!company) return <div className="text-gray-500">Empresa no encontrada</div>;

  const companyDeals = negocios?.filter((d) => d.company_id === company.id) ?? [];

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <button onClick={() => router.back()} className="text-blue-600 hover:underline text-sm">
          ← Volver
        </button>
        {canManage && (
          <Button
            onClick={() => {
              setDialogOpen(true);
            }}
          >
            Editar
          </Button>
        )}
      </div>

      <div className="bg-white rounded-lg border p-4 mb-4">
        <h2 className="text-xl font-semibold mb-3">{company.name}</h2>
        <dl className="grid grid-cols-2 gap-3 text-sm">
          <div><dt className="text-gray-500">NIT</dt><dd>{company.nit || "-"}</dd></div>
          <div><dt className="text-gray-500">Sector</dt><dd>{company.sector || "-"}</dd></div>
          <div><dt className="text-gray-500">Ciudad</dt><dd>{company.ciudad || "-"}</dd></div>
          <div><dt className="text-gray-500">Tipo</dt><dd>{company.tipo_empresa || "-"}</dd></div>
          <div><dt className="text-gray-500">Contactos</dt><dd>{company.total_contactos ?? "-"}</dd></div>
          <div><dt className="text-gray-500">Negocios</dt><dd>{company.total_negocios ?? "-"}</dd></div>
        </dl>
        <TagPicker entityType="company" entityId={company.id} />
      </div>

      <div className="bg-white rounded-lg border p-4">
        <h3 className="font-semibold mb-2">Negocios asociados</h3>
        {companyDeals.length > 0 ? (
          <ul className="divide-y">
            {companyDeals.map((d) => (
              <li
                key={d.id}
                className="py-2 text-sm flex justify-between cursor-pointer hover:underline text-blue-600"
                onClick={() => router.push(`/dashboard/crm?view=negocios&id=${d.id}`)}
              >
                <span>{d.nombre}</span>
                <span className="text-gray-500">{d.estado}</span>
              </li>
            ))}
          </ul>
        ) : (
          <p className="text-sm text-gray-400">Sin negocios asociados</p>
        )}
      </div>

      <CompanyDialog open={dialogOpen} onOpenChange={setDialogOpen} company={company} />
    </div>
  );
}
