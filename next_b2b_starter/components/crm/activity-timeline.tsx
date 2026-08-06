"use client";

import { useActivitiesQuery } from "@/lib/hooks/queries/use-crm-queries";
import { useCreateActivity } from "@/lib/hooks/mutations/use-crm-mutations";
import { useState } from "react";

export function ActivityTimeline() {
  const { data: activities, isLoading } = useActivitiesQuery();
  const createActivity = useCreateActivity();
  const [showForm, setShowForm] = useState(false);
  const [tipo, setTipo] = useState("nota");
  const [asunto, setAsunto] = useState("");
  const [contenido, setContenido] = useState("");

  const handleSubmit = () => {
    createActivity.mutate({ tipo, asunto, contenido } as any);
    setShowForm(false);
    setAsunto("");
    setContenido("");
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
        <button
          onClick={() => setShowForm(!showForm)}
          className="bg-blue-600 text-white px-4 py-2 rounded text-sm hover:bg-blue-700"
        >
          {showForm ? "Cancelar" : "Nueva actividad"}
        </button>
      </div>

      {showForm && (
        <div className="bg-gray-50 rounded-lg p-4 mb-6 border">
          <select value={tipo} onChange={(e) => setTipo(e.target.value)} className="border rounded px-3 py-2 w-full mb-2">
            <option value="nota">Nota</option>
            <option value="llamada">Llamada</option>
            <option value="correo">Correo</option>
            <option value="reunion">Reunión</option>
            <option value="tarea">Tarea</option>
          </select>
          <input
            type="text" placeholder="Asunto" value={asunto}
            onChange={(e) => setAsunto(e.target.value)}
            className="border rounded px-3 py-2 w-full mb-2"
          />
          <textarea
            placeholder="Contenido" value={contenido}
            onChange={(e) => setContenido(e.target.value)}
            className="border rounded px-3 py-2 w-full mb-2"
            rows={3}
          />
          <button onClick={handleSubmit} className="bg-blue-600 text-white px-4 py-2 rounded text-sm">
            Guardar
          </button>
        </div>
      )}

      <div className="space-y-3">
        {activities?.map((a) => (
          <div key={a.id} className="flex gap-3 p-3 bg-white rounded border">
            <div className="text-xl">{tipoIcon[a.tipo] || "📌"}</div>
            <div>
              <div className="font-medium text-sm">{a.asunto || a.tipo}</div>
              {a.contenido && <div className="text-sm text-gray-600 mt-1">{a.contenido}</div>}
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
