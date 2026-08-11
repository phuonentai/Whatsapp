"use client";

import { useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { usePipelinesQuery } from "@/lib/hooks/queries/use-crm-queries";
import {
  useCreatePipeline,
  useCreateStage,
  useUpdateStage,
} from "@/lib/hooks/mutations/use-crm-mutations";
import type { PipelineStageDto } from "@/lib/api/api/dto/crm.dto";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ErrorState } from "@/components/common/error-state";

interface StageRow {
  nombre: string;
  color: string;
}

function NewPipelineDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) {
  const createPipeline = useCreatePipeline();
  const createStage = useCreateStage();
  const saving = createPipeline.isPending || createStage.isPending;
  const [nombre, setNombre] = useState("");
  const [rows, setRows] = useState<StageRow[]>([{ nombre: "", color: "#3B82F6" }]);

  const addRow = () => setRows((prev) => [...prev, { nombre: "", color: "#3B82F6" }]);
  const updateRow = (index: number, patch: Partial<StageRow>) =>
    setRows((prev) => prev.map((r, i) => (i === index ? { ...r, ...patch } : r)));

  const reset = () => {
    setNombre("");
    setRows([{ nombre: "", color: "#3B82F6" }]);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!nombre.trim()) {
      toast.error("El nombre del pipeline es requerido");
      return;
    }
    let pipelineId: number | null = null;
    try {
      const created = await createPipeline.mutateAsync({ nombre });
      pipelineId = created.id;
      for (const [index, row] of rows.entries()) {
        await createStage.mutateAsync({
          pipelineId: created.id,
          data: { nombre: row.nombre, orden: index + 1, color: row.color || "#3B82F6" },
        });
      }
      toast.success("Pipeline creado");
      onOpenChange(false);
      reset();
    } catch {
      if (pipelineId !== null) {
        toast.error("Pipeline creado pero hubo un error al agregar las etapas. Inténtalo de nuevo.");
      } else {
        toast.error("Error al crear el pipeline");
      }
    }
  };

  return (
    <Dialog open={open} onOpenChange={(next) => !saving && onOpenChange(next)}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Nuevo pipeline</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="nombre">Nombre del pipeline</Label>
            <Input id="nombre" name="nombre" value={nombre} onChange={(e) => setNombre(e.target.value)} placeholder="Pipeline de Ventas" />
          </div>
          <div className="space-y-2">
            <Label>Etapas</Label>
            {rows.map((row, index) => (
              <div key={index} className="flex gap-2">
                <Input
                  name="stage_name"
                  placeholder="Nombre de la etapa"
                  value={row.nombre}
                  onChange={(e) => updateRow(index, { nombre: e.target.value })}
                />
                <Input
                  name="stage_color"
                  type="color"
                  value={row.color}
                  onChange={(e) => updateRow(index, { color: e.target.value })}
                  className="w-12 px-1 py-1"
                />
                {rows.length > 1 && (
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => setRows((prev) => prev.filter((_, i) => i !== index))}
                  >
                    Quitar
                  </Button>
                )}
              </div>
            ))}
            <Button type="button" variant="outline" size="sm" onClick={addRow}>
              Agregar Etapa
            </Button>
          </div>
          <DialogFooter className="gap-2 sm:gap-0">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={saving}>
              Cancelar
            </Button>
            <Button type="submit" disabled={saving}>
              {saving ? "Guardando..." : "Guardar"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function StageEditRow({ pipelineId, stage, onDone }: { pipelineId: number; stage: PipelineStageDto; onDone: () => void }) {
  const updateStage = useUpdateStage();
  const { register, handleSubmit } = useForm({
    defaultValues: {
      stage_name: stage.nombre,
      stage_color: stage.color ?? "#3B82F6",
      probabilidad: stage.probabilidad != null ? String(stage.probabilidad) : "",
    },
  });

  const onSubmit = handleSubmit(async (values) => {
    try {
      await updateStage.mutateAsync({
        pipelineId,
        stageId: stage.id,
        data: {
          nombre: values.stage_name,
          color: values.stage_color,
          probabilidad: values.probabilidad ? Number(values.probabilidad) : undefined,
        },
      });
      toast.success("Etapa actualizada");
      onDone();
    } catch {
      // error toast handled by mutation
    }
  });

  return (
    <form onSubmit={onSubmit} className="mt-2 flex items-center gap-2">
      <Input className="flex-1" placeholder="Nombre" {...register("stage_name")} />
      <Input type="color" className="w-12 px-1 py-1" {...register("stage_color")} />
      <Input className="w-24" placeholder="Prob. (%)" {...register("probabilidad")} />
      <Button type="submit" size="sm">Guardar</Button>
      <Button type="button" size="sm" variant="outline" onClick={onDone}>Cancelar</Button>
    </form>
  );
}

export function PipelineView() {
  const { data: pipelines, isLoading, isError, refetch, isRefetching } = usePipelinesQuery();
  const [newOpen, setNewOpen] = useState(false);
  const [editing, setEditing] = useState<{ pipelineId: number; stage: PipelineStageDto } | null>(null);

  if (isLoading) return <div className="text-gray-500">Cargando pipelines...</div>;

  if (isError) {
    return (
      <ErrorState
        title="Error al cargar los pipelines"
        description="No se pudieron cargar los pipelines. Inténtalo de nuevo."
        onRetry={() => refetch()}
        isRetrying={isRefetching}
      />
    );
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold">Pipelines</h2>
        <Button onClick={() => setNewOpen(true)}>Nuevo pipeline</Button>
      </div>

      <div data-testid="pipeline-list" className="space-y-6">
        {(!pipelines || pipelines.length === 0) && (
          <div className="text-gray-400 text-center py-8">No hay pipelines</div>
        )}

        {pipelines?.map((pipeline) => (
          <div key={pipeline.id} className="border rounded-lg p-4">
            <div className="flex items-center justify-between mb-3">
              <h3 className="font-medium">{pipeline.nombre}</h3>
              {pipeline.es_predeterminado && (
                <span className="text-xs bg-blue-100 text-blue-800 px-2 py-1 rounded">Predeterminado</span>
              )}
            </div>
            <div className="space-y-2">
              {pipeline.etapas
                .slice()
                .sort((a, b) => a.orden - b.orden)
                .map((stage) => (
                  <div key={stage.id} data-testid="stage-item" className="flex items-center gap-3 p-2 bg-gray-50 rounded border">
                    <div className="w-4 h-4 rounded-full" style={{ backgroundColor: stage.color || "#ccc" }} />
                    <span className="font-medium flex-1">{stage.nombre}</span>
                    <span className="text-sm text-gray-500">
                      {stage.probabilidad != null ? `${stage.probabilidad}%` : "Salida"}
                    </span>
                    <button
                      aria-label="Editar"
                      onClick={() => setEditing({ pipelineId: pipeline.id, stage })}
                      className="text-blue-600 hover:underline text-sm"
                    >
                      Editar
                    </button>
                  </div>
                ))}
            </div>
            {editing && editing.pipelineId === pipeline.id && (
              <StageEditRow
                pipelineId={editing.pipelineId}
                stage={editing.stage}
                onDone={() => setEditing(null)}
              />
            )}
          </div>
        ))}
      </div>

      <NewPipelineDialog open={newOpen} onOpenChange={setNewOpen} />
    </div>
  );
}
