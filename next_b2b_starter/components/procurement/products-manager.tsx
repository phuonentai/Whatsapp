"use client";

import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { ui } from "@/lib/copy/ui";
import { useProductsQuery } from "@/lib/hooks/queries/use-procurement-queries";
import { useCreateProduct, useUpdateProduct } from "@/lib/hooks/mutations/use-procurement-mutations";
import type { ProductDto } from "@/lib/api/api/dto/procurement.dto";
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

interface ProductFormValues {
  name: string;
  sku: string;
  unit: string;
}

export function ProductsManager() {
  const { data: products, isLoading, isError, refetch, isRefetching } = useProductsQuery();
  const createMutation = useCreateProduct();
  const updateMutation = useUpdateProduct();
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<ProductDto | null>(null);

  const { register, handleSubmit, reset } = useForm<ProductFormValues>();

  useEffect(() => {
    if (open) {
      reset({
        name: editing?.name ?? "",
        sku: editing?.sku ?? "",
        unit: editing?.unit ?? "und",
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

  const onSubmit = handleSubmit(async (values) => {
    if (!values.name.trim() || !values.sku.trim()) {
      toast.error("El nombre y el SKU son obligatorios");
      return;
    }
    try {
      if (editing) {
        await updateMutation.mutateAsync({
          id: editing.id,
          data: {
            name: values.name.trim(),
            sku: values.sku.trim(),
            unit: values.unit.trim() || "und",
            is_active: editing.is_active,
          },
        });
      } else {
        await createMutation.mutateAsync({
          name: values.name.trim(),
          sku: values.sku.trim(),
          unit: values.unit.trim() || "und",
        });
      }
      toast.success(ui.procurement.productSaved);
      setOpen(false);
    } catch {
      toast.error(ui.common.unexpectedError);
    }
  });

  const toggleActive = async (p: ProductDto) => {
    try {
      await updateMutation.mutateAsync({
        id: p.id,
        data: { name: p.name, sku: p.sku, unit: p.unit, is_active: !p.is_active },
      });
      toast.success(ui.procurement.productSaved);
    } catch {
      toast.error(ui.common.unexpectedError);
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">{ui.procurement.productsTitle}</h2>
        <Button onClick={() => { setEditing(null); setOpen(true); }}>
          {ui.procurement.addProduct}
        </Button>
      </div>

      {!products || products.length === 0 ? (
        <p className="text-gray-500">{ui.procurement.productsEmpty}</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{ui.procurement.productName}</TableHead>
              <TableHead>{ui.procurement.sku}</TableHead>
              <TableHead>{ui.procurement.unit}</TableHead>
              <TableHead>{ui.procurement.runStatus}</TableHead>
              <TableHead className="text-right">{ui.common.cancel}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {products.map((p) => (
              <TableRow key={p.id}>
                <TableCell className="font-medium">{p.name}</TableCell>
                <TableCell>{p.sku}</TableCell>
                <TableCell>{p.unit}</TableCell>
                <TableCell>
                  {p.is_active ? (
                    <Badge variant="default">{ui.procurement.active}</Badge>
                  ) : (
                    <Badge variant="secondary">{ui.procurement.inactive}</Badge>
                  )}
                </TableCell>
                <TableCell className="text-right space-x-2">
                  <Button variant="outline" size="sm" onClick={() => { setEditing(p); setOpen(true); }}>
                    {ui.common.save}
                  </Button>
                  <Button variant="ghost" size="sm" onClick={() => toggleActive(p)}>
                    {p.is_active ? ui.procurement.deactivate : ui.procurement.activate}
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
            <DialogTitle>{editing ? ui.procurement.editProduct : ui.procurement.addProduct}</DialogTitle>
          </DialogHeader>
          <form onSubmit={onSubmit} className="space-y-4">
            <div>
              <Label htmlFor="product-name">{ui.procurement.productName}</Label>
              <Input id="product-name" {...register("name")} placeholder="Papel carta" />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="product-sku">{ui.procurement.sku}</Label>
                <Input id="product-sku" {...register("sku")} placeholder="PAP-001" />
              </div>
              <div>
                <Label htmlFor="product-unit">{ui.procurement.unit}</Label>
                <Input id="product-unit" {...register("unit")} placeholder="resma" />
              </div>
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
