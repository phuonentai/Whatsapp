"use client";

import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import { companySchema, type CompanyFormValues } from "@/lib/crm/validation";
import { useCreateCompany, useUpdateCompany } from "@/lib/hooks/mutations/use-crm-mutations";
import type { CompanyDto } from "@/lib/api/api/dto/crm.dto";
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

interface CompanyDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  company?: CompanyDto | null;
}

export function CompanyDialog({ open, onOpenChange, company }: CompanyDialogProps) {
  const isEdit = Boolean(company);
  const createMutation = useCreateCompany();
  const updateMutation = useUpdateCompany();
  const saving = createMutation.isPending || updateMutation.isPending;

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<CompanyFormValues>({
    resolver: zodResolver(companySchema),
    defaultValues: { name: "", nit: "", sector: "", ciudad: "" },
  });

  useEffect(() => {
    if (open) {
      reset({
        name: company?.name ?? "",
        nit: company?.nit ?? "",
        sector: company?.sector ?? "",
        ciudad: company?.ciudad ?? "",
      });
    }
  }, [open, company, reset]);

  const onSubmit = handleSubmit(async (values) => {
    try {
      if (isEdit && company) {
        await updateMutation.mutateAsync({ id: company.id, data: values });
        toast.success("Empresa actualizada");
      } else {
        await createMutation.mutateAsync(values);
        toast.success("Empresa creada");
      }
      onOpenChange(false);
    } catch {
      // dialog stays open with entered values; error toast handled by mutation
    }
  });

  return (
    <Dialog open={open} onOpenChange={(next) => !saving && onOpenChange(next)}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{isEdit ? "Editar empresa" : "Nueva empresa"}</DialogTitle>
        </DialogHeader>
        <form onSubmit={onSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">Nombre</Label>
            <Input id="name" placeholder="Nombre de la empresa" {...register("name")} />
            {errors.name && <p className="text-sm text-red-600">{errors.name.message}</p>}
          </div>
          <div className="space-y-2">
            <Label htmlFor="nit">NIT</Label>
            <Input id="nit" placeholder="NIT" {...register("nit")} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="sector">Sector</Label>
            <Input id="sector" placeholder="Sector industrial" {...register("sector")} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="ciudad">Ciudad</Label>
            <Input id="ciudad" placeholder="Ciudad" {...register("ciudad")} />
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
