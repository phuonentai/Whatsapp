"use client";

import { useState } from "react";
import { format } from "date-fns";
import { AlertCircle, ChevronLeft, ChevronRight } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useActivitiesQuery } from "@/lib/hooks/queries/use-crm-queries";

const TIPO_OPTIONS = [
  { value: "", label: "Todos" },
  { value: "nota", label: "Nota" },
  { value: "llamada", label: "Llamada" },
  { value: "correo", label: "Correo" },
  { value: "reunion", label: "Reunión" },
  { value: "tarea", label: "Tarea" },
  { value: "whatsapp_message", label: "WhatsApp" },
  { value: "sistema", label: "Sistema" },
] as const;

const PAGE_SIZE = 20;

export function AuditLogView() {
  const [tipo, setTipo] = useState("");
  const [offset, setOffset] = useState(0);

  const { data: activities, isLoading, error } = useActivitiesQuery({
    tipo: tipo || undefined,
    limit: PAGE_SIZE,
    offset,
  });

  const resetPagination = () => setOffset(0);

  const handleTipoChange = (next: string) => {
    setTipo(next);
    resetPagination();
  };

  if (error) {
    return (
      <div className="flex items-start gap-3 rounded-xl border border-red-200 bg-red-50 p-4">
        <AlertCircle className="mt-0.5 h-5 w-5 shrink-0 text-red-600" />
        <div>
          <p className="text-sm font-medium text-red-900">Failed to load audit log</p>
          <p className="text-sm text-red-700">
            {error instanceof Error ? error.message : "An unexpected error occurred."}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-sm text-gray-600">
          Read-only record of activity across your organization.
        </p>
        <select
          aria-label="Filter audit log by type"
          value={tipo}
          onChange={(e) => handleTipoChange(e.target.value)}
          className="w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm sm:w-auto"
        >
          {TIPO_OPTIONS.map((o) => (
            <option key={o.value} value={o.value}>{o.label}</option>
          ))}
        </select>
      </div>

      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="h-20 animate-pulse rounded-xl border border-gray-100 bg-gray-50" />
          ))}
        </div>
      ) : !activities?.length ? (
        <div className="rounded-xl border border-gray-200 bg-gray-50 p-8 text-center text-sm text-gray-500">
          No activity found{ tipo ? " for this type" : "" }.
        </div>
      ) : (
        <div className="overflow-hidden rounded-xl border border-gray-200 bg-white">
          <ul className="divide-y divide-gray-100" data-testid="audit-log-list">
            {activities.map((a) => (
              <li key={a.id} className="flex flex-col gap-1 px-4 py-3 sm:flex-row sm:items-start sm:justify-between">
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="inline-flex items-center rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-700">
                      {tipoLabel(a.tipo)}
                    </span>
                    {a.asunto && (
                      <span className="truncate text-sm font-semibold text-gray-900">
                        {a.asunto}
                      </span>
                    )}
                  </div>
                  {a.contenido && (
                    <p className="mt-1 line-clamp-2 text-sm text-gray-600">{a.contenido}</p>
                  )}
                  <p className="mt-1 text-xs text-gray-400">
                    {format(new Date(a.realizada_en), "MMM d, yyyy 'at' h:mm a")}
                    {a.realizada_por_nombre ? ` • ${a.realizada_por_nombre}` : " • System"}
                  </p>
                </div>
                {entityRefs(a) && (
                  <p className="shrink-0 text-xs text-gray-400">{entityRefs(a)}</p>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}

      <div className="flex items-center justify-between">
        <Button
          variant="outline"
          size="sm"
          disabled={offset === 0 || isLoading}
          onClick={() => setOffset((o) => Math.max(0, o - PAGE_SIZE))}
        >
          <ChevronLeft className="mr-1 h-4 w-4" />
          Previous
        </Button>
        <Button
          variant="outline"
          size="sm"
          disabled={isLoading || (activities?.length ?? 0) < PAGE_SIZE}
          onClick={() => setOffset((o) => o + PAGE_SIZE)}
        >
          Next
          <ChevronRight className="ml-1 h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}

function tipoLabel(tipo: string): string {
  const match = TIPO_OPTIONS.find((o) => o.value === tipo);
  return match?.label ?? tipo;
}

function entityRefs(a: { contact_id?: number; company_id?: number; deal_id?: number; conversation_id?: number }): string | null {
  const refs: string[] = [];
  if (a.contact_id) refs.push(`Contacto #${a.contact_id}`);
  if (a.company_id) refs.push(`Empresa #${a.company_id}`);
  if (a.deal_id) refs.push(`Negocio #${a.deal_id}`);
  if (a.conversation_id) refs.push(`Conversación #${a.conversation_id}`);
  return refs.length ? refs.join(" • ") : null;
}
