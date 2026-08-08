"use client";

import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import { contactSchema, LEAD_STATUSES, type ContactFormValues } from "@/lib/crm/validation";
import { useCreateContact, useUpdateContact } from "@/lib/hooks/mutations/use-crm-mutations";
import type { ContactDto } from "@/lib/api/api/dto/crm.dto";
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

const LEAD_STATUS_LABELS: Record<string, string> = {
  nuevo: "Nuevo",
  contactado: "Contactado",
  calificado: "Calificado",
  descalificado: "Descalificado",
  cliente: "Cliente",
};

interface ContactDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  contact?: ContactDto | null;
}

export function ContactDialog({ open, onOpenChange, contact }: ContactDialogProps) {
  const isEdit = Boolean(contact);
  const createMutation = useCreateContact();
  const updateMutation = useUpdateContact();
  const saving = createMutation.isPending || updateMutation.isPending;

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<ContactFormValues>({
    resolver: zodResolver(contactSchema),
    defaultValues: { phone: "", display_name: "", email: "", lead_status: "nuevo" },
  });

  useEffect(() => {
    if (open) {
      reset({
        phone: contact?.phone_number ?? "",
        display_name: contact?.display_name ?? "",
        email: contact?.email ?? "",
        lead_status: (LEAD_STATUSES as readonly string[]).includes(contact?.lead_status ?? "")
          ? (contact!.lead_status as ContactFormValues["lead_status"])
          : "nuevo",
      });
    }
  }, [open, contact, reset]);

  const onSubmit = handleSubmit(async (values) => {
    const payload = {
      phone_number: values.phone,
      display_name: values.display_name ?? "",
      email: values.email ?? "",
      lead_status: values.lead_status,
    };
    if (isEdit && contact) {
      await updateMutation.mutateAsync({ id: contact.id, data: payload });
      toast.success("Contacto actualizado");
    } else {
      await createMutation.mutateAsync(payload);
      toast.success("Contacto creado");
    }
    onOpenChange(false);
  });

  return (
    <Dialog open={open} onOpenChange={(next) => !saving && onOpenChange(next)}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{isEdit ? "Editar contacto" : "Nuevo contacto"}</DialogTitle>
        </DialogHeader>
        <form onSubmit={onSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="phone">Teléfono</Label>
            <Input id="phone" placeholder="+573001234567" {...register("phone")} />
            {errors.phone && <p className="text-sm text-red-600">{errors.phone.message}</p>}
          </div>
          <div className="space-y-2">
            <Label htmlFor="display_name">Nombre</Label>
            <Input id="display_name" placeholder="Nombre del contacto" {...register("display_name")} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="email">Correo</Label>
            <Input id="email" type="email" placeholder="correo@ejemplo.com" {...register("email")} />
            {errors.email && <p className="text-sm text-red-600">{errors.email.message}</p>}
          </div>
          <div className="space-y-2">
            <Label htmlFor="lead_status">Estado</Label>
            <select
              id="lead_status"
              className="border rounded px-3 py-2 w-full"
              {...register("lead_status")}
            >
              {LEAD_STATUSES.map((status) => (
                <option key={status} value={status}>
                  {LEAD_STATUS_LABELS[status]}
                </option>
              ))}
            </select>
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
