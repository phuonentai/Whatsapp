"use client";
import { useCompaniesQuery } from "@/lib/hooks/queries/use-crm-queries";
export function CompanyTable() {
  const { data: companies, isLoading } = useCompaniesQuery();
  if (isLoading) return <div className="text-gray-500">Cargando empresas...</div>;
  return (
    <div>
      <table className="w-full border-collapse">
        <thead><tr className="bg-gray-100">
          <th className="text-left p-2">Nombre</th><th className="text-left p-2">NIT</th>
          <th className="text-left p-2">Sector</th><th className="text-left p-2">Ciudad</th>
          <th className="text-left p-2">Tipo</th><th className="text-left p-2">Contactos</th>
        </tr></thead>
        <tbody>
          {companies?.map((c) => (
            <tr key={c.id} className="border-b hover:bg-gray-50">
              <td className="p-2 font-medium">{c.name}</td>
              <td className="p-2">{c.nit || "-"}</td>
              <td className="p-2">{c.sector || "-"}</td>
              <td className="p-2">{c.ciudad || "-"}</td>
              <td className="p-2">{c.tipo_empresa || "-"}</td>
              <td className="p-2">{c.total_contactos || 0}</td>
            </tr>
          ))}
          {(!companies || companies.length === 0) && (
            <tr><td colSpan={6} className="p-4 text-center text-gray-400">No hay empresas</td></tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
