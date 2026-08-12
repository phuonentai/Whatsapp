"use client";

import { useActivitiesQuery } from "@/lib/hooks/queries/use-crm-queries";
import { useCreateActivity } from "@/lib/hooks/mutations/use-crm-mutations";
import { usePermissions } from "@/lib/hooks/use-permissions";
import { crmRepository } from "@/lib/api/api/repositories/crm-repository";
import { ErrorState } from "@/components/common/error-state";
import { Skeleton } from "@/components/ui/skeleton";
import { useState } from "react";
import { Download } from "lucide-react";
import { useCsvExport } from "@/lib/csv-export";

const TIPO_OPTIONS = [
  { value: "nota", label: "Nota" },
  { value: "llamada", label: "Llamada" },
  { value: "correo", label: "Correo" },
  { value: "reunion", label: "Reunión" },
  { value: "tarea", label: "Tarea" },
] as const;

export function ActivityTimeline() {
  const [tipoFilter, setTipoFilter] = useState("");
  const { data: activities, isLoading, isError, refetch, isRefetching } = useActivitiesQuery({ tipo: tipoFilter || undefined });
  const createActivity = useCreateActivity();
  const [showForm, setShowForm] = useState(false);
  const [tipo, setTipo] = useState("nota");
  const [asunto, setAsunto] = useState("");
  const [contenido, setContenido] = useState("");
  const [estado, setEstado] = useState("pendiente");
  const [fechaVencimiento, setFechaVencimiento] = useState("");

  const isTarea = tipo === "tarea";

  const { hasPermission } = usePermissions();
  const canExport = hasPermission("activity:export");
  const { isExporting, handleExport } = useCsvExport({
    run: () => crmRepository.exportActivities(),
    successMessage: "Actividades exportadas",
    errorMessage: "Error al exportar actividades",
  });

  const handleSubmit = () => {
    createActivity.mutate({
      tipo,
      asunto,
      contenido,
      ...(isTarea ? { estado, fecha_vencimiento: fechaVencimiento || undefined } : {}),
    });
    setShowForm(false);
    setAsunto("");
    setContenido("");
    setEstado("pendiente");
    setFechaVencimiento("");
  };

  const tipoIcon: Record<string, string> = {
    nota: "📝", llamada: "📞", correo: "📧", reunion: "🤝",
    tarea: "✅", whatsapp_message: "💬", sistema: "⚙️",
  };

  if (isLoading) {
    return (
      <div className="space-y-3" data-testid="activity-timeline" aria-busy="true">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="flex gap-3 p-3 bg-white rounded border">
            <Skeleton className="h-8 w-8 rounded" />
            <div className="flex-1 space-y-2">
              <Skeleton className="h-4 w-40" />
              <Skeleton className="h-3 w-full" />
              <Skeleton className="h-3 w-24" />
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (isError) {
    return (
      <ErrorState
        title="Error al cargar las actividades"
        description="No se pudieron cargar las actividades. Inténtalo de nuevo."
        onRetry={() => refetch()}
        isRetrying={isRefetching}
      />
    );
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-lg font-semibold">Actividad</h2>
        <div className="flex items-center gap-2">
          <select
            data-testid="activity-type-filter"
            aria-label="Filtrar por tipo"
            value={tipoFilter}
            onChange={(e) => setTipoFilter(e.target.value)}
            className="border rounded px-3 py-2 text-sm"
          >
            <option value="">Todos</option>
            {TIPO_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>{o.label}</option>
            ))}
          </select>
          <button
            onClick={() => setShowForm(!showForm)}
            className="bg-emerald-500 text-white px-4 py-2 rounded text-sm hover:bg-emerald-600"
          >
            {showForm ? "Cancelar" : "Nueva actividad"}
          </button>
          {canExport && (
            <button
              onClick={handleExport}
              disabled={isExporting}
              className="border border-gray-300 text-gray-700 px-4 py-2 rounded text-sm hover:bg-gray-50 flex items-center gap-2"
            >
              <Download className="h-4 w-4" />
              {isExporting ? "Exportando..." : "Exportar"}
            </button>
          )}
        </div>
      </div>

      {showForm && (
        <div className="bg-gray-50 rounded-lg p-4 mb-6 border">
          <select name="tipo" value={tipo} onChange={(e) => setTipo(e.target.value)} className="border rounded px-3 py-2 w-full mb-2">
            {TIPO_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>{o.label}</option>
            ))}
          </select>
          <input
            name="asunto"
            type="text" placeholder="Asunto" value={asunto}
            onChange={(e) => setAsunto(e.target.value)}
            className="border rounded px-3 py-2 w-full mb-2"
          />
          <textarea
            name="contenido"
            placeholder="Contenido" value={contenido}
            onChange={(e) => setContenido(e.target.value)}
            className="border rounded px-3 py-2 w-full mb-2"
            rows={3}
          />
          {isTarea && (
            <div className="flex gap-2 mb-2">
              <input
                type="date"
                aria-label="Fecha de vencimiento"
                value={fechaVencimiento}
                onChange={(e) => setFechaVencimiento(e.target.value)}
                className="border rounded px-3 py-2"
              />
              <select
                aria-label="Estado"
                value={estado}
                onChange={(e) => setEstado(e.target.value)}
                className="border rounded px-3 py-2"
              >
                <option value="pendiente">Pendiente</option>
                <option value="completada">Completada</option>
              </select>
            </div>
          )}
          <button onClick={handleSubmit} className="bg-emerald-500 text-white px-4 py-2 rounded text-sm">
            Guardar
          </button>
        </div>
      )}

      <div className="space-y-3" data-testid="activity-timeline">
        {activities?.map((a) => (
          <div key={a.id} data-testid="activity-item" className="flex gap-3 p-3 bg-white rounded border">
            <div className="text-xl">{tipoIcon[a.tipo] || "📌"}</div>
            <div>
              <div className="font-medium text-sm">{a.asunto || a.tipo}</div>
              {a.contenido && <div className="text-sm text-gray-600 mt-1">{a.contenido}</div>}
              {a.fecha_vencimiento && (
                <div className="text-xs text-gray-500 mt-1">
                  Vence: {new Date(a.fecha_vencimiento).toLocaleDateString("es-CO")}
                  {a.estado ? ` • ${a.estado}` : ""}
                </div>
              )}
              <div className="text-xs text-gray-400 mt-1">
                {new Date(a.realizada_en).toLocaleDateString("es-CO", {
                  day: "numeric", month: "long", hour: "2-digit", minute: "2-digit",
                })}
                {a.realizada_por_nombre && ` • ${a.realizada_por_nombre}`}
              </div>
            </div>
          </div>
        ))}
        {(!activities || activities.length === 0) && (
          <div className="text-center text-gray-400 py-8">Sin actividad registrada</div>
        )}
      </div>
    </div>
  );
}
