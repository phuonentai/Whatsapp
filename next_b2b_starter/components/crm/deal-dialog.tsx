"use client";

import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import { dealSchema, type DealFormValues } from "@/lib/crm/validation";
import { useCreateDeal, useUpdateDeal } from "@/lib/hooks/mutations/use-crm-mutations";
import { useCompaniesQuery } from "@/lib/hooks/queries/use-crm-queries";
import { useContactsQuery } from "@/lib/hooks/queries/use-crm-queries";
import type { DealDto, PipelineStageDto } from "@/lib/api/api/dto/crm.dto";
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

interface DealDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  deal?: DealDto | null;
  pipelineId: number;
  stages: PipelineStageDto[];
}

export function DealDialog({ open, onOpenChange, deal, pipelineId, stages }: DealDialogProps) {
  const isEdit = Boolean(deal);
  const createMutation = useCreateDeal();
  const updateMutation = useUpdateDeal();
  const saving = createMutation.isPending || updateMutation.isPending;
  const { data: companies } = useCompaniesQuery();
  const { data: contacts } = useContactsQuery();

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<DealFormValues>({
    resolver: zodResolver(dealSchema),
    defaultValues: {
      nombre: "",
      monto: "",
      moneda: "COP",
      company_id: "",
      contact_id: "",
      stage_id: "",
    },
  });

  useEffect(() => {
    if (open) {
      reset({
        nombre: deal?.nombre ?? "",
        monto: deal?.monto != null ? String(deal.monto) : "",
        moneda: deal?.moneda ?? "COP",
        company_id: deal?.company_id != null ? String(deal.company_id) : "",
        contact_id: deal?.contact_id != null ? String(deal.contact_id) : "",
        stage_id: deal?.stage_id != null ? String(deal.stage_id) : "",
      });
    }
  }, [open, deal, reset]);

  const onSubmit = handleSubmit(async (values) => {
    const payload = {
      nombre: values.nombre,
      moneda: values.moneda || "COP",
      monto: values.monto ? Number(values.monto) : undefined,
      company_id: values.company_id ? Number(values.company_id) : undefined,
      contact_id: values.contact_id ? Number(values.contact_id) : undefined,
    };
    if (isEdit && deal) {
      await updateMutation.mutateAsync({ id: deal.id, data: payload });
      toast.success("Negocio actualizado");
    } else {
      await createMutation.mutateAsync({ ...payload, pipeline_id: pipelineId, stage_id: values.stage_id ? Number(values.stage_id) : undefined });
      toast.success("Negocio creado");
    }
    onOpenChange(false);
  });

  return (
    <Dialog open={open} onOpenChange={(next) => !saving && onOpenChange(next)}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{isEdit ? "Editar negocio" : "Nuevo negocio"}</DialogTitle>
        </DialogHeader>
        <form onSubmit={onSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="nombre">Nombre</Label>
            <Input id="nombre" placeholder="Nombre del negocio" {...register("nombre")} />
            {errors.nombre && <p className="text-sm text-red-600">{errors.nombre.message}</p>}
          </div>
          <div className="space-y-2">
            <Label htmlFor="monto">Monto</Label>
            <Input id="monto" type="number" placeholder="5000000" {...register("monto")} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="moneda">Moneda</Label>
            <Input id="moneda" placeholder="COP" {...register("moneda")} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="company_id">Empresa</Label>
            <select id="company_id" className="border rounded px-3 py-2 w-full" {...register("company_id")}>
              <option value="">Sin empresa</option>
              {companies?.map((c) => (
                <option key={c.id} value={c.id}>{c.name}</option>
              ))}
            </select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="contact_id">Contacto</Label>
            <select id="contact_id" className="border rounded px-3 py-2 w-full" {...register("contact_id")}>
              <option value="">Sin contacto</option>
              {contacts?.map((c) => (
                <option key={c.id} value={c.id}>{c.display_name || c.phone_number}</option>
              ))}
            </select>
          </div>
          {!isEdit && (
            <div className="space-y-2">
              <Label htmlFor="stage_id">Etapa</Label>
              <select id="stage_id" className="border rounded px-3 py-2 w-full" {...register("stage_id")}>
                <option value="">Primera etapa</option>
                {stages.map((s) => (
                  <option key={s.id} value={s.id}>{s.nombre}</option>
                ))}
              </select>
            </div>
          )}
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
