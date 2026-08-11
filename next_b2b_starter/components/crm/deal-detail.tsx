"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useDealQuery, useDealActivitiesQuery, usePipelinesQuery } from "@/lib/hooks/queries/use-crm-queries";
import { useFeature } from "@/lib/hooks/use-entitlement";
import { DealDialog } from "@/components/crm/deal-dialog";
import { TagPicker } from "@/components/crm/tag-picker";
import { ErrorState } from "@/components/common/error-state";
import { Button } from "@/components/ui/button";

export function DealDetail({ id }: { id: number }) {
  const router = useRouter();
  const { data: deal, isLoading, isError, refetch, isRefetching } = useDealQuery(id);
  const { data: activities } = useDealActivitiesQuery(id);
  const { data: pipelines } = usePipelinesQuery();
  const canManage = useFeature("crm_deals");
  const [dialogOpen, setDialogOpen] = useState(false);

  if (isLoading) return <div className="text-gray-500">Cargando negocio...</div>;

  if (isError) {
    return (
      <ErrorState
        title="Error al cargar el negocio"
        description="No se pudo cargar el negocio. Inténtalo de nuevo."
        onRetry={() => refetch()}
        isRetrying={isRefetching}
      />
    );
  }

  if (!deal) return <div className="text-gray-500">Negocio no encontrado</div>;

  const stage = pipelines
    ?.flatMap((p) => p.etapas)
    .find((s) => s.id === deal.stage_id);
  const dealPipeline = pipelines?.find((p) => p.id === deal.pipeline_id);

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
        <h2 className="text-xl font-semibold mb-3">{deal.nombre}</h2>
        <dl className="grid grid-cols-2 gap-3 text-sm">
          <div>
            <dt className="text-gray-500">Monto</dt>
            <dd>
              {deal.monto != null ? `${deal.monto.toLocaleString("es-CO")} ${deal.moneda || "COP"}` : "-"}
            </dd>
          </div>
          <div><dt className="text-gray-500">Etapa</dt><dd>{stage?.nombre || "Sin etapa"}</dd></div>
          <div><dt className="text-gray-500">Estado</dt><dd>{deal.estado}</dd></div>
          <div><dt className="text-gray-500">Contacto</dt><dd>{deal.contact_name || "-"}</dd></div>
          <div><dt className="text-gray-500">Empresa</dt><dd>{deal.company_name || "-"}</dd></div>
          {deal.fecha_cierre_esperada && (
            <div>
              <dt className="text-gray-500">Cierre esperado</dt>
              <dd>{new Date(deal.fecha_cierre_esperada).toLocaleDateString("es-CO")}</dd>
            </div>
          )}
        </dl>
        {deal.notas && (
          <div className="mt-3 text-sm">
            <dt className="text-gray-500">Notas</dt>
            <dd className="mt-1">{deal.notas}</dd>
          </div>
        )}
        <TagPicker entityType="deal" entityId={deal.id} />
      </div>

      <div className="bg-white rounded-lg border p-4">
        <h3 className="font-semibold mb-2">Actividad</h3>
        <div className="space-y-2">
          {activities?.map((a) => (
            <div key={a.id} className="text-sm border-b pb-2">
              <div className="font-medium">{a.asunto || a.tipo}</div>
              {a.contenido && <div className="text-gray-600">{a.contenido}</div>}
              <div className="text-xs text-gray-400">
                {new Date(a.realizada_en).toLocaleDateString("es-CO", {
                  day: "numeric", month: "long", hour: "2-digit", minute: "2-digit",
                })}
                {a.realizada_por_nombre && ` • ${a.realizada_por_nombre}`}
              </div>
            </div>
          ))}
          {(!activities || activities.length === 0) && (
            <p className="text-sm text-gray-400">Sin actividad registrada</p>
          )}
        </div>
      </div>

      <DealDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        deal={deal}
        pipelineId={deal.pipeline_id}
        stages={dealPipeline?.etapas ?? []}
      />
    </div>
  );
}
