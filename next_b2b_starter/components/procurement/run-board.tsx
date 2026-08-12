"use client";

import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "next/navigation";
import { toast } from "sonner";
import { ui } from "@/lib/copy/ui";
import {
  useRunBoardQuery,
  useRunOrdersQuery,
  useProductsQuery,
} from "@/lib/hooks/queries/use-procurement-queries";
import { usePlaceOrder } from "@/lib/hooks/mutations/use-procurement-mutations";
import type { BoardDto, OrderDto, ResponseItemDto } from "@/lib/api/api/dto/procurement.dto";
import { ErrorState } from "@/components/common/error-state";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Checkbox } from "@/components/ui/checkbox";
import { runStatusBadge } from "./run-wizard";

interface OrderLine {
  key: number;
  extractedName: string;
  productId: number;
  quantity: number;
}

interface OrderDraft {
  supplierId: number;
  supplierName: string;
  lines: OrderLine[];
  notes: string;
  override: boolean;
}

function orderStateBadge(order: OrderDto) {
  switch (order.status) {
    case "confirm_sent":
      return <Badge variant="default">Confirmación enviada</Badge>;
    case "send_blocked":
      return <Badge variant="destructive">Envío bloqueado ({order.blocked_reason ?? ""})</Badge>;
    default:
      return <Badge variant="secondary">{order.status}</Badge>;
  }
}

/** Best product match by extracted product name (case/accents-insensitive). */
function matchProduct(products: { id: number; name: string }[] | undefined, name: string) {
  const norm = (s: string) => s.toLowerCase().normalize("NFD").replace(/[\u0300-\u036f]/g, "");
  const target = norm(name);
  return (products ?? []).find((p) => norm(p.name).includes(target) || target.includes(norm(p.name)))?.id ?? 0;
}

export function RunBoard({ runId }: { runId: number }) {
  const searchParams = useSearchParams();
  const { data: board, isLoading, isError, refetch, isRefetching } = useRunBoardQuery(runId);
  const { data: orders } = useRunOrdersQuery(runId);
  const { data: products } = useProductsQuery();
  const placeOrder = usePlaceOrder();
  const [draft, setDraft] = useState<OrderDraft | null>(null);

  useEffect(() => {
    if (searchParams.get("run") !== String(runId) && searchParams.get("run")) {
      const url = new URL(window.location.href);
      url.searchParams.set("run", String(runId));
      window.history.replaceState(null, "", url.toString());
    }
  }, [runId, searchParams]);

  const answeredRows = useMemo(
    () => (board?.rows ?? []).filter((r) => r.recipient_status === "answered"),
    [board]
  );

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

  const openOrder = (board: BoardDto, row: BoardDto["rows"][number]) => {
    const items = row.response?.items ?? [];
    const lines: OrderLine[] = items.map((it: ResponseItemDto, idx: number) => ({
      key: idx,
      extractedName: it.product_name,
      productId: matchProduct(products, it.product_name),
      quantity: it.cantidad_disponible ?? 1,
    }));
    setDraft({
      supplierId: row.supplier_id,
      supplierName: row.display_name || row.nit,
      lines,
      notes: "",
      override: false,
    });
  };

  const submitOrder = async () => {
    if (!draft) return;
    const valid = draft.lines.filter((l) => l.productId > 0 && l.quantity > 0);
    if (valid.length === 0) {
      toast.error(ui.procurement.orderItems + " (selecciona productos y cantidades)");
      return;
    }
    try {
      await placeOrder.mutateAsync({
        runId,
        data: {
          supplier_id: draft.supplierId,
          items: valid.map((l) => ({ product_id: l.productId, quantity: l.quantity })),
          notes: draft.notes.trim() || null,
          override: draft.override,
        },
      });
      toast.success(ui.procurement.orderPlaced);
      setDraft(null);
    } catch {
      toast.error(ui.procurement.orderBlocked);
    }
  };

  const updateLine = (key: number, patch: Partial<OrderLine>) => {
    setDraft((d) =>
      d ? { ...d, lines: d.lines.map((l) => (l.key === key ? { ...l, ...patch } : l)) } : d
    );
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">{ui.procurement.boardTitle}</h2>
        {board?.run && runStatusBadge(board.run.status)}
      </div>

      {board?.summary && (
        <Card>
          <CardHeader>
            <CardTitle>{ui.procurement.boardSummary}</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-gray-700">{board.summary}</p>
          </CardContent>
        </Card>
      )}

      {!board || board.rows.length === 0 ? (
        <p className="text-gray-500">{ui.procurement.runsEmpty}</p>
      ) : (
        <Card>
          <CardContent className="p-0">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b text-left text-gray-500">
                  <th className="p-3">{ui.procurement.selectSuppliers}</th>
                  <th className="p-3">{ui.procurement.recipientStatus}</th>
                  <th className="p-3">{ui.procurement.availability}</th>
                  <th className="p-3">{ui.procurement.unitPrice}</th>
                  <th className="p-3">{ui.procurement.leadTime}</th>
                  <th className="p-3">{ui.procurement.resumen}</th>
                  <th className="p-3 text-right">{ui.procurement.placeOrder}</th>
                </tr>
              </thead>
              <tbody>
                {board.rows.map((row) => {
                  const items = row.response?.items ?? [];
                  const available = items.filter((i) => i.disponible).length;
                  const minPrice = items
                    .map((i) => i.precio_unitario)
                    .filter((p): p is number => p != null)
                    .sort((a, b) => a - b)[0];
                  const requiresHuman = row.response?.requiere_humano ?? false;
                  const orderExists = (orders ?? []).some((o) => o.supplier_id === row.supplier_id);
                  return (
                    <tr key={row.recipient_id} className="border-b">
                      <td className="p-3 font-medium">{row.display_name || row.nit}</td>
                      <td className="p-3">
                        {row.recipient_status === "answered" ? (
                          <Badge variant="default">{row.recipient_status}</Badge>
                        ) : (
                          <Badge variant="secondary">{row.recipient_status}</Badge>
                        )}
                      </td>
                      <td className="p-3">
                        {items.length > 0 ? `${available}/${items.length}` : "—"}
                      </td>
                      <td className="p-3">
                        {minPrice != null ? `$ ${minPrice.toLocaleString("es-CO")} ${ui.procurement.cop}` : "—"}
                      </td>
                      <td className="p-3">{items.map((i) => i.tiempo_entrega).find(Boolean) ?? "—"}</td>
                      <td className="p-3 text-gray-600">
                        {row.response?.resumen ?? ui.procurement.noResponse}
                        {requiresHuman && (
                          <div>
                            <Badge variant="destructive">{ui.procurement.requiresHuman}</Badge>
                          </div>
                        )}
                      </td>
                      <td className="p-3 text-right">
                        {orderExists ? (
                          orderStateBadge((orders ?? []).find((o) => o.supplier_id === row.supplier_id)!)
                        ) : row.recipient_status === "answered" ? (
                          <Button size="sm" onClick={() => openOrder(board, row)}>
                            {requiresHuman ? ui.procurement.placeOrderOverride : ui.procurement.placeOrder}
                          </Button>
                        ) : (
                          <span className="text-gray-400">—</span>
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </CardContent>
        </Card>
      )}

      <Dialog open={draft !== null} onOpenChange={(v) => !v && setDraft(null)}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>
              {ui.procurement.placeOrder}: {draft?.supplierName}
            </DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <Label>{ui.procurement.orderItems}</Label>
              <div className="mt-2 space-y-2">
                {(draft?.lines ?? []).map((line) => (
                  <div key={line.key} className="flex items-center gap-2 text-sm">
                    <span className="w-40 truncate">{line.extractedName}</span>
                    <Select
                      value={String(line.productId)}
                      onValueChange={(v) => updateLine(line.key, { productId: Number(v) })}
                    >
                      <SelectTrigger className="w-44">
                        <SelectValue placeholder="Producto…" />
                      </SelectTrigger>
                      <SelectContent>
                        {(products ?? []).map((p) => (
                          <SelectItem key={p.id} value={String(p.id)}>
                            {p.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <Input
                      type="number"
                      min={1}
                      className="w-24"
                      value={line.quantity}
                      onChange={(e) => updateLine(line.key, { quantity: Number(e.target.value) })}
                    />
                  </div>
                ))}
                {(!draft || draft.lines.length === 0) && (
                  <p className="text-sm text-gray-500">{ui.procurement.noResponse}</p>
                )}
              </div>
            </div>
            <div>
              <Label>{ui.procurement.orderNotes}</Label>
              <Input
                value={draft?.notes ?? ""}
                onChange={(e) => setDraft((d) => (d ? { ...d, notes: e.target.value } : d))}
              />
            </div>
            <label className="flex items-center gap-2 text-sm">
              <Checkbox
                checked={draft?.override ?? false}
                onCheckedChange={(v) => setDraft((d) => (d ? { ...d, override: v === true } : d))}
              />
              {ui.procurement.placeOrderOverride}
            </label>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDraft(null)}>
              {ui.common.cancel}
            </Button>
            <Button onClick={submitOrder} disabled={placeOrder.isPending}>
              {ui.procurement.placeOrder}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {answeredRows.length === 0 && (
        <p className="text-sm text-gray-400">{ui.procurement.noResponse}</p>
      )}
    </div>
  );
}
