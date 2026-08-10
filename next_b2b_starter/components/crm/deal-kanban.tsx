"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import {
  DndContext,
  PointerSensor,
  useDraggable,
  useDroppable,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import { usePipelinesQuery, useDealsQuery } from "@/lib/hooks/queries/use-crm-queries";
import { useMoveDealStage, useDeleteDeal } from "@/lib/hooks/mutations/use-crm-mutations";
import { useFeature } from "@/lib/hooks/use-entitlement";
import { usePermissions } from "@/lib/hooks/use-permissions";
import type { DealDto, PipelineStageDto } from "@/lib/api/api/dto/crm.dto";
import { crmRepository } from "@/lib/api/api/repositories/crm-repository";
import { DealDialog } from "@/components/crm/deal-dialog";
import { ConfirmDialog } from "@/components/crm/confirm-dialog";
import { Button } from "@/components/ui/button";
import { Download } from "lucide-react";

interface DealCardProps {
  deal: DealDto;
  stage: PipelineStageDto;
  allStages: PipelineStageDto[];
  onOpen: (deal: DealDto) => void;
  onEdit: (deal: DealDto) => void;
  onDelete: (deal: DealDto) => void;
  onMove: (dealId: number, stageId: number, oldName: string, newName: string) => void;
}

function DealCard({ deal, stage, allStages, onOpen, onEdit, onDelete, onMove }: DealCardProps) {
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({ id: deal.id });
  const style = transform
    ? { transform: `translate3d(${transform.x}px, ${transform.y}px, 0)` }
    : undefined;

  return (
    <div
      ref={setNodeRef}
      data-testid="deal-card"
      className={`bg-white rounded border p-3 mb-2 shadow-sm cursor-pointer ${isDragging ? "opacity-50" : ""}`}
      style={style}
      onClick={() => onOpen(deal)}
    >
      <div className="flex items-center justify-between gap-2">
        <div className="font-medium text-sm text-blue-600 hover:underline">{deal.nombre}</div>
        <button
          aria-label="Arrastrar negocio"
          className="text-gray-400 text-sm cursor-grab active:cursor-grabbing"
          {...attributes}
          {...listeners}
        >
          ⋮⋮
        </button>
      </div>
      {deal.monto != null && (
        <div className="text-sm text-gray-600 mt-1">
          ${deal.monto.toLocaleString("es-CO")} {deal.moneda}
        </div>
      )}
      {deal.company_name && (
        <div className="text-xs text-gray-400 mt-1">{deal.company_name}</div>
      )}
      <div className="mt-2 flex items-center gap-2">
        <select
          className="text-xs border rounded px-2 py-1 flex-1"
          defaultValue=""
          onClick={(e) => e.stopPropagation()}
          onChange={(e) => {
            const target = allStages.find((s) => s.id === Number(e.target.value));
            if (target && target.id !== stage.id) {
              onMove(deal.id, target.id, stage.nombre, target.nombre);
            }
            e.currentTarget.value = "";
          }}
        >
          <option value="" disabled>Mover a...</option>
          {allStages.filter((s) => s.id !== stage.id).map((s) => (
            <option key={s.id} value={s.id}>{s.nombre}</option>
          ))}
        </select>
        <button onClick={(e) => { e.stopPropagation(); onEdit(deal); }} className="text-blue-600 hover:underline text-xs">
          Editar
        </button>
        <button
          aria-label="Eliminar"
          onClick={(e) => { e.stopPropagation(); onDelete(deal); }}
          className="text-red-600 hover:underline text-xs"
        >
          Eliminar
        </button>
      </div>
    </div>
  );
}

function StageColumn({
  stage,
  deals,
  allStages,
  onOpen,
  onEdit,
  onDelete,
  onMove,
}: {
  stage: PipelineStageDto;
  deals: DealDto[];
  allStages: PipelineStageDto[];
  onOpen: (deal: DealDto) => void;
  onEdit: (deal: DealDto) => void;
  onDelete: (deal: DealDto) => void;
  onMove: (dealId: number, stageId: number, oldName: string, newName: string) => void;
}) {
  const { setNodeRef, isOver } = useDroppable({ id: stage.id });
  return (
    <div
      ref={setNodeRef}
      data-testid="stage-column"
      className={`min-w-[250px] bg-gray-50 rounded-lg p-3 ${isOver ? "ring-2 ring-blue-400" : ""}`}
    >
      <div className="flex items-center gap-2 mb-3">
        <div className="w-3 h-3 rounded-full" style={{ backgroundColor: stage.color || "#ccc" }} />
        <h3 className="font-medium text-sm">{stage.nombre}</h3>
        <span className="text-xs text-gray-400 ml-auto">{deals.length}</span>
      </div>
      {deals.map((deal) => (
        <DealCard
          key={deal.id}
          deal={deal}
          stage={stage}
          allStages={allStages}
          onOpen={onOpen}
          onEdit={onEdit}
          onDelete={onDelete}
          onMove={onMove}
        />
      ))}
      {deals.length === 0 && (
        <div className="text-xs text-gray-400 text-center py-4">Sin negocios</div>
      )}
    </div>
  );
}

export function DealKanban() {
  const router = useRouter();
  const { data: pipelines } = usePipelinesQuery();
  const defaultPipeline = useMemo(
    () => pipelines?.find((p) => p.es_predeterminado) ?? pipelines?.[0],
    [pipelines]
  );
  const [selectedPipelineId, setSelectedPipelineId] = useState<number | undefined>(undefined);
  const pipeline = pipelines?.find((p) => p.id === selectedPipelineId) ?? defaultPipeline;
  const { data: deals } = useDealsQuery({ pipeline_id: pipeline?.id });
  const moveStage = useMoveDealStage();
  const deleteMutation = useDeleteDeal();
  const canManage = useFeature("crm_deals");
  const { hasPermission } = usePermissions();
  const canExport = hasPermission("deal:export");
  const [isExporting, setIsExporting] = useState(false);
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 5 } }));

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<DealDto | null>(null);
  const [deleting, setDeleting] = useState<DealDto | null>(null);

  const stages = pipeline?.etapas || [];

  const stageById = useMemo(() => new Map(stages.map((s) => [s.id, s])), [stages]);

  const handleMove = (dealId: number, stageId: number, oldName: string, newName: string) => {
    moveStage.mutate({
      id: dealId,
      data: { stage_id: stageId, old_stage_name: oldName, new_stage_name: newName },
    });
  };

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over) return;
    const deal = deals?.find((d) => d.id === active.id);
    const targetStage = stageById.get(over.id as number);
    const sourceStage = deal ? stageById.get(deal.stage_id ?? -1) : undefined;
    if (!deal || !targetStage || targetStage.id === deal.stage_id) return;
    handleMove(deal.id, targetStage.id, sourceStage?.nombre ?? "", targetStage.nombre);
  };

  const handleDelete = async () => {
    if (!deleting) return;
    await deleteMutation.mutateAsync(deleting.id);
    toast.success("Negocio eliminado");
    setDeleting(null);
  };

  const handleExport = async () => {
    setIsExporting(true);
    try {
      await crmRepository.exportDeals();
      toast.success("Negocios exportados");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Error al exportar negocios");
    } finally {
      setIsExporting(false);
    }
  };

  return (
    <div>
      <div className="flex items-center gap-4 mb-4">
        <h2 className="text-lg font-semibold">Negocios</h2>
        <label className="text-sm text-gray-600">
          Pipeline:
          <select
            className="ml-2 border rounded px-3 py-2"
            value={pipeline?.id ?? ""}
            onChange={(e) => setSelectedPipelineId(Number(e.target.value) || undefined)}
          >
            {pipelines?.map((p) => (
              <option key={p.id} value={p.id}>{p.nombre}</option>
            ))}
          </select>
        </label>
        {canManage && (
          <Button
            onClick={() => {
              setEditing(null);
              setDialogOpen(true);
            }}
          >
            Nuevo negocio
          </Button>
        )}
        {canManage && canExport && (
          <Button variant="outline" onClick={handleExport} disabled={isExporting}>
            <Download className="mr-2 h-4 w-4" />
            {isExporting ? "Exportando..." : "Exportar"}
          </Button>
        )}
      </div>

      <DndContext sensors={sensors} onDragEnd={handleDragEnd}>
        <div data-testid="kanban-board" className="flex gap-4 overflow-x-auto pb-4">
          {stages.map((stage) => (
            <StageColumn
              key={stage.id}
              stage={stage}
              deals={deals?.filter((d) => d.stage_id === stage.id) || []}
              allStages={stages}
              onOpen={(deal) => router.push(`/dashboard/crm?view=negocios&id=${deal.id}`)}
              onEdit={(deal) => {
                setEditing(deal);
                setDialogOpen(true);
              }}
              onDelete={setDeleting}
              onMove={handleMove}
            />
          ))}
        </div>
      </DndContext>

      <DealDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        deal={editing}
        pipelineId={pipeline?.id ?? 0}
        stages={stages}
      />
      <ConfirmDialog
        open={Boolean(deleting)}
        onOpenChange={(next) => !next && setDeleting(null)}
        title="Eliminar negocio"
        description={`¿Estás seguro de eliminar el negocio "${deleting?.nombre}"? Esta acción no se puede deshacer.`}
        confirmLabel="Eliminar"
        loading={deleteMutation.isPending}
        onConfirm={handleDelete}
      />
    </div>
  );
}
