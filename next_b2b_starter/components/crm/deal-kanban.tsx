"use client";

import { usePipelinesQuery, useDealsQuery } from "@/lib/hooks/queries/use-crm-queries";
import { useMoveDealStage } from "@/lib/hooks/mutations/use-crm-mutations";

export function DealKanban() {
  const { data: pipelines } = usePipelinesQuery();
  const { data: deals } = useDealsQuery();
  const moveStage = useMoveDealStage();
  const stages = pipelines?.[0]?.etapas || [];

  const handleMove = (dealId: number, stageId: number, oldName: string, newName: string) => {
    moveStage.mutate({ id: dealId, data: { stage_id: stageId, old_stage_name: oldName, new_stage_name: newName } });
  };

  return (
    <div>
      <h2 className="text-lg font-semibold mb-4">Pipeline de Ventas</h2>
      <div className="flex gap-4 overflow-x-auto pb-4">
        {stages.map((stage) => (
          <div key={stage.id} className="min-w-[250px] bg-gray-50 rounded-lg p-3">
            <div className="flex items-center gap-2 mb-3">
              <div className="w-3 h-3 rounded-full" style={{ backgroundColor: stage.color || "#ccc" }} />
              <h3 className="font-medium text-sm">{stage.nombre}</h3>
              <span className="text-xs text-gray-400 ml-auto">
                {deals?.filter((d) => d.stage_id === stage.id).length || 0}
              </span>
            </div>
            {deals
              ?.filter((d) => d.stage_id === stage.id)
              .map((deal) => (
                <div key={deal.id} className="bg-white rounded border p-3 mb-2 shadow-sm">
                  <div className="font-medium text-sm">{deal.nombre}</div>
                  {deal.monto && (
                    <div className="text-sm text-gray-600 mt-1">
                      ${deal.monto.toLocaleString("es-CO")} {deal.moneda}
                    </div>
                  )}
                  {deal.company_name && (
                    <div className="text-xs text-gray-400 mt-1">{deal.company_name}</div>
                  )}
                  <div className="mt-2">
                    <select
                      className="text-xs border rounded px-2 py-1 w-full"
                      onChange={(e) => {
                        const targetStage = stages.find((s) => s.id === Number(e.target.value));
                        if (targetStage) handleMove(deal.id, targetStage.id, stage.nombre, targetStage.nombre);
                      }}
                      defaultValue=""
                    >
                      <option value="" disabled>Mover a...</option>
                      {stages.filter((s) => s.id !== stage.id).map((s) => (
                        <option key={s.id} value={s.id}>{s.nombre}</option>
                      ))}
                    </select>
                  </div>
                </div>
              ))}
            {deals?.filter((d) => d.stage_id === stage.id).length === 0 && (
              <div className="text-xs text-gray-400 text-center py-4">Sin negocios</div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
