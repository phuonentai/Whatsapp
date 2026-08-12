"use client";

import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { ui } from "@/lib/copy/ui";
import { useSuppliersQuery } from "@/lib/hooks/queries/use-procurement-queries";
import {
  useCreateSupplier,
  useUpdateSupplier,
} from "@/lib/hooks/mutations/use-procurement-mutations";
import type { SupplierDto } from "@/lib/api/api/dto/procurement.dto";
import { ErrorState } from "@/components/common/error-state";
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
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

interface SupplierFormValues {
  nit: string;
  phone: string;
  display_name: string;
  delivery_days?: string;
  min_order_amount?: string;
  notes?: string;
}

export function SuppliersManager() {
  const { data: suppliers, isLoading, isError, refetch, isRefetching } = useSuppliersQuery();
  const createMutation = useCreateSupplier();
  const updateMutation = useUpdateSupplier();
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<SupplierDto | null>(null);

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<SupplierFormValues>();

  useEffect(() => {
    if (open) {
      reset({
        nit: editing?.nit ?? "",
        phone: "",
        display_name: "",
        delivery_days: editing?.delivery_days != null ? String(editing.delivery_days) : "",
        min_order_amount: editing?.min_order_amount != null ? String(editing.min_order_amount) : "",
        notes: editing?.notes ?? "",
      });
    }
  }, [open, editing, reset]);

  if (isLoading) return <div className="text-gray-500">{ui.procurement.loading}</div>;
  if (isError) {
    return (
      <ErrorState
        title={ui.procurement.errorLoading}
        description=""
        onRetry={() => refetch()}
        isRetrying={isRefetching}
      />
    );
  }

  const openCreate = () => {
    setEditing(null);
    setOpen(true);
  };
  const openEdit = (s: SupplierDto) => {
    setEditing(s);
    setOpen(true);
  };

  const onSubmit = handleSubmit(async (values) => {
    const payload = {
      nit: values.nit.trim(),
      phone: values.phone.trim(),
      display_name: values.display_name.trim(),
      delivery_days: values.delivery_days ? Number(values.delivery_days) : null,
      min_order_amount: values.min_order_amount ? Number(values.min_order_amount) : null,
      notes: values.notes?.trim() || null,
    };
    try {
      if (editing) {
        await updateMutation.mutateAsync({
          id: editing.id,
          data: {
            delivery_days: payload.delivery_days,
            min_order_amount: payload.min_order_amount,
            notes: payload.notes,
            is_active: editing.is_active,
          },
        });
      } else {
        if (!payload.nit || !payload.phone) {
          toast.error("El NIT y el teléfono son obligatorios");
          return;
        }
        await createMutation.mutateAsync(payload);
      }
      toast.success(ui.procurement.supplierSaved);
      setOpen(false);
    } catch {
      toast.error(ui.procurement.supplierExists);
    }
  });

  const toggleActive = async (s: SupplierDto) => {
    try {
      await updateMutation.mutateAsync({
        id: s.id,
        data: {
          delivery_days: s.delivery_days,
          min_order_amount: s.min_order_amount,
          notes: s.notes,
          is_active: !s.is_active,
        },
      });
      toast.success(ui.procurement.supplierSaved);
    } catch {
      toast.error(ui.common.unexpectedError);
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">{ui.procurement.suppliersTitle}</h2>
        <Button onClick={openCreate}>{ui.procurement.addSupplier}</Button>
      </div>

      {!suppliers || suppliers.length === 0 ? (
        <p className="text-gray-500">{ui.procurement.suppliersEmpty}</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{ui.procurement.displayName}</TableHead>
              <TableHead>{ui.procurement.nit}</TableHead>
              <TableHead>{ui.procurement.phone}</TableHead>
              <TableHead>{ui.procurement.deliveryDays}</TableHead>
              <TableHead>{ui.procurement.minOrderAmount}</TableHead>
              <TableHead>{ui.procurement.runStatus}</TableHead>
              <TableHead className="text-right">{ui.common.cancel}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {suppliers.map((s) => (
              <TableRow key={s.id}>
                <TableCell className="font-medium">{s.display_name || s.nit}</TableCell>
                <TableCell>{s.nit}</TableCell>
                <TableCell>{s.phone_number}</TableCell>
                <TableCell>{s.delivery_days ?? "—"}</TableCell>
                <TableCell>{s.min_order_amount != null ? `$ ${s.min_order_amount}` : "—"}</TableCell>
                <TableCell>
                  {s.is_active ? (
                    <Badge variant="default">{ui.procurement.active}</Badge>
                  ) : (
                    <Badge variant="secondary">{ui.procurement.inactive}</Badge>
                  )}
                </TableCell>
                <TableCell className="text-right space-x-2">
                  <Button variant="outline" size="sm" onClick={() => openEdit(s)}>
                    {ui.common.save}
                  </Button>
                  <Button variant="ghost" size="sm" onClick={() => toggleActive(s)}>
                    {s.is_active ? ui.procurement.deactivate : ui.procurement.activate}
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{editing ? ui.procurement.editSupplier : ui.procurement.addSupplier}</DialogTitle>
          </DialogHeader>
          <form onSubmit={onSubmit} className="space-y-4">
            {!editing && (
              <>
                <div>
                  <Label htmlFor="supplier-display-name">{ui.procurement.displayName}</Label>
                  <Input id="supplier-display-name" {...register("display_name")} placeholder="Distribuidora Andina S.A.S." />
                </div>
                <div>
                  <Label htmlFor="supplier-nit">{ui.procurement.nit}</Label>
                  <Input id="supplier-nit" {...register("nit")} placeholder="901234567" />
                  {errors.nit && <p className="text-sm text-red-600">{errors.nit.message}</p>}
                </div>
                <div>
                  <Label htmlFor="supplier-phone">{ui.procurement.phone}</Label>
                  <Input id="supplier-phone" {...register("phone")} placeholder="+573001234567" />
                </div>
              </>
            )}
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="supplier-delivery-days">{ui.procurement.deliveryDays}</Label>
                <Input id="supplier-delivery-days" type="number" {...register("delivery_days")} />
              </div>
              <div>
                <Label htmlFor="supplier-min-amount">{ui.procurement.minOrderAmount}</Label>
                <Input id="supplier-min-amount" type="number" {...register("min_order_amount")} />
              </div>
            </div>
            <div>
              <Label>{ui.procurement.notes}</Label>
              <Input {...register("notes")} />
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setOpen(false)}>
                {ui.common.cancel}
              </Button>
              <Button type="submit" disabled={createMutation.isPending || updateMutation.isPending}>
                {ui.common.save}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
