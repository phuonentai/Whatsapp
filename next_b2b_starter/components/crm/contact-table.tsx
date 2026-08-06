"use client";

import { useContactsQuery } from "@/lib/hooks/queries/use-crm-queries";
import { useState } from "react";

export function ContactTable() {
  const [search, setSearch] = useState("");
  const [filterStatus, setFilterStatus] = useState("");
  const { data: contacts, isLoading } = useContactsQuery({ lead_status: filterStatus || undefined });

  if (isLoading) return <div className="text-gray-500">Cargando contactos...</div>;

  return (
    <div>
      <div className="flex gap-4 mb-4">
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
          </tr>
        </thead>
        <tbody>
          {contacts?.map((c) => (
            <tr key={c.id} className="border-b hover:bg-gray-50">
              <td className="p-2 font-medium">{c.display_name || c.phone_number}</td>
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
            </tr>
          ))}
          {(!contacts || contacts.length === 0) && (
            <tr><td colSpan={7} className="p-4 text-center text-gray-400">No hay contactos</td></tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
