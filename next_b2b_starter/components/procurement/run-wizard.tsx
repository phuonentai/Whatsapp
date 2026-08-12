"use client";

import { useState } from "react";
import { toast } from "sonner";
import { ui } from "@/lib/copy/ui";
import {
  useRunsQuery,
  useSuppliersQuery,
  useProductsQuery,
} from "@/lib/hooks/queries/use-procurement-queries";
import { useCreateRun, useSendRun } from "@/lib/hooks/mutations/use-procurement-mutations";
import type { RunStatus } from "@/lib/api/api/dto/procurement.dto";
import { ErrorState } from "@/components/common/error-state";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export const RUN_STATUS_LABELS: Record<RunStatus, string> = {
  draft: ui.procurement.statusDraft,
  sending: ui.procurement.statusSending,
  awaiting_responses: ui.procurement.statusAwaitingResponses,
  completed: ui.procurement.statusCompleted,
  partially_answered: ui.procurement.statusPartiallyAnswered,
  failed: ui.procurement.statusFailed,
  escalated: ui.procurement.statusEscalated,
  cancelled: ui.procurement.statusCancelled,
};

export function runStatusBadge(status: RunStatus) {
  switch (status) {
    case "completed":
      return <Badge variant="default">{RUN_STATUS_LABELS[status]}</Badge>;
    case "escalated":
    case "failed":
      return <Badge variant="destructive">{RUN_STATUS_LABELS[status]}</Badge>;
    case "awaiting_responses":
      return <Badge variant="default">{RUN_STATUS_LABELS[status]}</Badge>;
    default:
      return <Badge variant="secondary">{RUN_STATUS_LABELS[status]}</Badge>;
  }
}

interface WizardState {
  supplierIds: number[];
  quantities: Record<number, string>;
  nota: string;
}

export function RunWizard() {
  const { data: runs, isLoading: runsLoading, isError, refetch, isRefetching } = useRunsQuery();
  const { data: suppliers } = useSuppliersQuery();
  const { data: products } = useProductsQuery();
  const createRun = useCreateRun();
  const sendRun = useSendRun();
  const [showWizard, setShowWizard] = useState(false);
  const [state, setState] = useState<WizardState>({ supplierIds: [], quantities: {}, nota: "" });

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

  const toggleSupplier = (id: number) => {
    setState((s) => ({
      ...s,
      supplierIds: s.supplierIds.includes(id)
        ? s.supplierIds.filter((x) => x !== id)
        : [...s.supplierIds, id],
    }));
  };

  const setQty = (productId: number, qty: string) => {
    setState((s) => ({ ...s, quantities: { ...s.quantities, [productId]: qty } }));
  };

  const handleCreate = async () => {
    if (state.supplierIds.length === 0) {
      toast.error(ui.procurement.selectSuppliers + " (obligatorio)");
      return;
    }
    const productsWithQty = (products ?? [])
      .filter((p) => state.quantities[p.id] !== undefined && Number(state.quantities[p.id]) > 0)
      .map((p) => ({ product_id: p.id, quantity: Number(state.quantities[p.id]) }));
    if (productsWithQty.length === 0) {
      toast.error(ui.procurement.selectProducts + " (obligatorio)");
      return;
    }
    try {
      const run = await createRun.mutateAsync({
        supplier_ids: state.supplierIds,
        products: productsWithQty,
        nota: state.nota.trim() || null,
      });
      if (run.status === "escalated") {
        toast.warning(ui.procurement.runEscalatedNotice);
      } else {
        toast.success(ui.procurement.runCreated);
      }
      setShowWizard(false);
      setState({ supplierIds: [], quantities: {}, nota: "" });
    } catch {
      toast.error(ui.procurement.runEscalatedNotice);
    }
  };

  const handleSend = async (runId: number) => {
    try {
      await sendRun.mutateAsync(runId);
      toast.success(ui.procurement.runSent);
    } catch {
      toast.error(ui.common.unexpectedError);
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">{ui.procurement.runsTitle}</h2>
        <Button onClick={() => setShowWizard((v) => !v)}>{ui.procurement.newRun}</Button>
      </div>

      {showWizard && (
        <Card>
          <CardHeader>
            <CardTitle>{ui.procurement.newRun}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <Label>{ui.procurement.selectSuppliers}</Label>
              <div className="mt-2 space-y-2">
                {(suppliers ?? []).map((s) => (
                  <label key={s.id} className="flex items-center gap-2 text-sm">
                    <Checkbox
                      checked={state.supplierIds.includes(s.id)}
                      onCheckedChange={() => toggleSupplier(s.id)}
                      disabled={!s.is_active}
                    />
                    {s.display_name || s.nit} ({s.nit})
                  </label>
                ))}
              </div>
            </div>
            <div>
              <Label>{ui.procurement.selectProducts}</Label>
              <div className="mt-2 space-y-2">
                {(products ?? []).map((p) => (
                  <div key={p.id} className="flex items-center gap-3 text-sm">
                    <span className="w-48 truncate">{p.name} ({p.sku})</span>
                    <Input
                      type="number"
                      min={0}
                      className="w-24"
                      placeholder={ui.procurement.quantity}
                      value={state.quantities[p.id] ?? ""}
                      onChange={(e) => setQty(p.id, e.target.value)}
                    />
                  </div>
                ))}
              </div>
            </div>
            <div>
              <Label>{ui.procurement.runNota}</Label>
              <Input
                placeholder={ui.procurement.runNotaPlaceholder}
                value={state.nota}
                onChange={(e) => setState((s) => ({ ...s, nota: e.target.value }))}
              />
            </div>
            <Button onClick={handleCreate} disabled={createRun.isPending}>
              {ui.procurement.createRun}
            </Button>
          </CardContent>
        </Card>
      )}

      {runsLoading ? (
        <p className="text-gray-500">{ui.procurement.loading}</p>
      ) : !runs || runs.length === 0 ? (
        <p className="text-gray-500">{ui.procurement.runsEmpty}</p>
      ) : (
        <div className="space-y-2">
          {runs.map((run) => (
            <div key={run.id} className="flex items-center justify-between rounded-lg border p-3">
              <div className="space-y-1">
                <div className="flex items-center gap-2">
                  <span className="font-medium">#{run.id}</span>
                  {runStatusBadge(run.status)}
                </div>
                {run.nota && <p className="text-sm text-gray-500">{run.nota}</p>}
                <p className="text-xs text-gray-400">
                  {ui.procurement.runCreatedAt}: {new Date(run.created_at).toLocaleString("es-CO")}
                </p>
              </div>
              <div className="flex items-center gap-2">
                <a
                  href={`/dashboard/procurement?run=${run.id}`}
                  className="text-sm font-medium text-blue-600 hover:underline"
                >
                  {ui.procurement.boardTitle}
                </a>
                {run.status === "draft" && (
                  <Button size="sm" onClick={() => handleSend(run.id)} disabled={sendRun.isPending}>
                    {ui.procurement.sendRun}
                  </Button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
