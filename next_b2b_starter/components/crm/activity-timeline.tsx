"use client";

import { useActivitiesQuery } from "@/lib/hooks/queries/use-crm-queries";
import { useCreateActivity } from "@/lib/hooks/mutations/use-crm-mutations";
import { useState } from "react";

const TIPO_OPTIONS = [
  { value: "nota", label: "Nota" },
  { value: "llamada", label: "Llamada" },
  { value: "correo", label: "Correo" },
  { value: "reunion", label: "Reunión" },
  { value: "tarea", label: "Tarea" },
] as const;

export function ActivityTimeline() {
  const [tipoFilter, setTipoFilter] = useState("");
  const { data: activities, isLoading } = useActivitiesQuery({ tipo: tipoFilter || undefined });
  const createActivity = useCreateActivity();
  const [showForm, setShowForm] = useState(false);
  const [tipo, setTipo] = useState("nota");
  const [asunto, setAsunto] = useState("");
  const [contenido, setContenido] = useState("");
  const [estado, setEstado] = useState("pendiente");
  const [fechaVencimiento, setFechaVencimiento] = useState("");

  const isTarea = tipo === "tarea";

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

  if (isLoading) return <div className="text-gray-500">Cargando actividades...</div>;

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
            className="bg-blue-600 text-white px-4 py-2 rounded text-sm hover:bg-blue-700"
          >
            {showForm ? "Cancelar" : "Nueva actividad"}
          </button>
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
          <button onClick={handleSubmit} className="bg-blue-600 text-white px-4 py-2 rounded text-sm">
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
