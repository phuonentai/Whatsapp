"use client";
import { usePipelinesQuery } from "@/lib/hooks/queries/use-crm-queries";
export function PipelineEditor() {
  const { data: pipelines, isLoading } = usePipelinesQuery();
  if (isLoading) return <div className="text-gray-500">Cargando pipelines...</div>;
  const stages = pipelines?.[0]?.etapas || [];
  return (
    <div>
      <h2 className="text-lg font-semibold mb-4">Pipeline de Ventas</h2>
      <div className="space-y-2">
        {stages.map((s, i) => (
          <div key={s.id} className="flex items-center gap-4 p-3 bg-white rounded border">
            <span className="text-gray-400 w-6">{i + 1}.</span>
            <div className="w-4 h-4 rounded-full" style={{ backgroundColor: s.color || "#ccc" }} />
            <span className="font-medium flex-1">{s.nombre}</span>
            <span className="text-sm text-gray-500">{s.probabilidad != null ? `${s.probabilidad}%` : "Salida"}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
