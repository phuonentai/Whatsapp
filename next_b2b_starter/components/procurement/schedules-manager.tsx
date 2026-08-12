"use client";

import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { ui } from "@/lib/copy/ui";
import {
  useSchedulesQuery,
  useScheduleDetailQuery,
  useSuppliersQuery,
  useProductsQuery,
} from "@/lib/hooks/queries/use-procurement-queries";
import {
  useCreateSchedule,
  useUpdateSchedule,
  usePauseSchedule,
  useResumeSchedule,
  useDeleteSchedule,
} from "@/lib/hooks/mutations/use-procurement-mutations";
import type {
  ProductDto,
  ScheduleDto,
  ScheduleStatusDto,
  SupplierDto,
} from "@/lib/api/api/dto/procurement.dto";
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

const WEEKDAYS: { value: number; label: string }[] = [
  { value: 0, label: ui.procurement.days.sunday },
  { value: 1, label: ui.procurement.days.monday },
  { value: 2, label: ui.procurement.days.tuesday },
  { value: 3, label: ui.procurement.days.wednesday },
  { value: 4, label: ui.procurement.days.thursday },
  { value: 5, label: ui.procurement.days.friday },
  { value: 6, label: ui.procurement.days.saturday },
];

interface ScheduleFormValues {
  name: string;
  run_time: string;
  note?: string;
}

const DAY_NAMES: Record<number, string> = Object.fromEntries(
  WEEKDAYS.map((d) => [d.value, d.label])
);

function formatNextRun(iso: string): string {
  try {
    return new Date(iso).toLocaleString("es-CO", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return iso;
  }
}

function daySummary(days: number[]): string {
  const sorted = [...days].sort((a, b) => a - b);
  if (sorted.length === 7) return "Todos los días";
  if (
    sorted.length === 5 &&
    sorted.every((d) => [1, 2, 3, 4, 5].includes(d))
  ) {
    return "Lun a Vie";
  }
  return sorted.map((d) => DAY_NAMES[d]?.slice(0, 3)).join(", ");
}

export function SchedulesManager() {
  const { data: schedules, isLoading, isError, refetch, isRefetching } = useSchedulesQuery();
  const { data: suppliers } = useSuppliersQuery();
  const { data: products } = useProductsQuery();
  const createMutation = useCreateSchedule();
  const updateMutation = useUpdateSchedule();
  const pauseMutation = usePauseSchedule();
  const resumeMutation = useResumeSchedule();
  const deleteMutation = useDeleteSchedule();

  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<ScheduleDto | null>(null);
  const [detailId, setDetailId] = useState<number | null>(null);

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
  const openEdit = (s: ScheduleDto) => {
    setEditing(s);
    setOpen(true);
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">{ui.procurement.schedulesTitle}</h2>
        <Button onClick={openCreate}>{ui.procurement.addSchedule}</Button>
      </div>

      {(schedules ?? []).length === 0 ? (
        <p className="text-sm text-gray-500">{ui.procurement.schedulesEmpty}</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{ui.procurement.scheduleName}</TableHead>
              <TableHead>{ui.procurement.runTime}</TableHead>
              <TableHead>{ui.procurement.daysOfWeek}</TableHead>
              <TableHead>{ui.procurement.nextRunAt}</TableHead>
              <TableHead>{ui.procurement.lastRunStatus}</TableHead>
              <TableHead>Estado</TableHead>
              <TableHead className="text-right">Acciones</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(schedules ?? []).map((entry: ScheduleStatusDto) => {
              const s = entry.Schedule;
              const pending = pauseMutation.isPending || resumeMutation.isPending;
              return (
                <TableRow key={s.ID}>
                  <TableCell className="font-medium">{s.Name}</TableCell>
                  <TableCell>{s.RunTime}</TableCell>
                  <TableCell>{daySummary(s.DaysOfWeek)}</TableCell>
                  <TableCell>{formatNextRun(s.NextRunAt)}</TableCell>
                  <TableCell>
                    {entry.HasLastRun ? (
                      <span className="text-sm">{entry.LastRunStatus}</span>
                    ) : (
                      <span className="text-sm text-gray-400">{ui.procurement.neverRun}</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <Badge variant={s.IsActive ? "default" : "secondary"}>
                      {s.IsActive ? ui.procurement.scheduleActive : ui.procurement.schedulePaused}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-2">
                      <Button variant="outline" size="sm" onClick={() => setDetailId(s.ID)}>
                        {ui.procurement.recentRuns}
                      </Button>
                      <Button variant="outline" size="sm" onClick={() => openEdit(s)}>
                        {ui.procurement.editSchedule}
                      </Button>
                      {s.IsActive ? (
                        <Button
                          variant="outline"
                          size="sm"
                          disabled={pending}
                          onClick={() => pauseMutation.mutate(s.ID)}
                        >
                          {ui.procurement.schedulePause}
                        </Button>
                      ) : (
                        <Button
                          variant="outline"
                          size="sm"
                          disabled={pending}
                          onClick={() => resumeMutation.mutate(s.ID)}
                        >
                          {ui.procurement.scheduleResume}
                        </Button>
                      )}
                      <Button
                        variant="destructive"
                        size="sm"
                        disabled={deleteMutation.isPending}
                        onClick={() => {
                          if (confirm(ui.procurement.scheduleDeleteConfirm)) {
                            deleteMutation.mutate(s.ID);
                          }
                        }}
                      >
                        {ui.procurement.scheduleDelete}
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      )}

      {open && (
        <ScheduleFormDialog
          key={editing?.ID ?? "new"}
          schedule={editing}
          suppliers={suppliers ?? []}
          products={products ?? []}
          onClose={() => setOpen(false)}
        />
      )}

      {detailId != null && <ScheduleDetailDialog scheduleId={detailId} onClose={() => setDetailId(null)} />}
    </div>
  );
}

function ScheduleDetailDialog({ scheduleId, onClose }: { scheduleId: number; onClose: () => void }) {
  const { data: detail, isLoading, isError, refetch, isRefetching } =
    useScheduleDetailQuery(scheduleId);
  const deleteMutation = useDeleteSchedule();
  const resumeMutation = useResumeSchedule();

  return (
    <Dialog open onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{ui.procurement.schedulesTitle}</DialogTitle>
        </DialogHeader>
        {isLoading && <div className="text-gray-500">{ui.procurement.loading}</div>}
        {isError && (
          <ErrorState
            title={ui.procurement.errorLoading}
            description=""
            onRetry={() => refetch()}
            isRetrying={isRefetching}
          />
        )}
        {detail && (
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-lg font-semibold">{detail.Schedule.Name}</p>
                <p className="text-sm text-gray-500">
                  {detail.Schedule.RunTime} · {daySummary(detail.Schedule.DaysOfWeek)}
                </p>
              </div>
              {detail.OverdueRecipients > 0 && (
                <Badge variant="destructive">
                  {detail.OverdueRecipients} {ui.procurement.overdueRecipients}
                </Badge>
              )}
            </div>
            <div className="flex flex-wrap gap-2">
              <Badge variant={detail.Schedule.IsActive ? "default" : "secondary"}>
                {detail.Schedule.IsActive ? ui.procurement.scheduleActive : ui.procurement.schedulePaused}
              </Badge>
              {!detail.Schedule.IsActive && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => resumeMutation.mutate(detail.Schedule.ID)}
                >
                  {ui.procurement.scheduleResume}
                </Button>
              )}
            </div>
            <div>
              <p className="text-sm font-medium">{ui.procurement.recentRuns}</p>
              <ul className="mt-2 space-y-1">
                {detail.RecentRuns.length === 0 && (
                  <li className="text-sm text-gray-400">{ui.procurement.neverRun}</li>
                )}
                {detail.RecentRuns.map((run) => (
                  <li key={run.ID} className="flex items-center justify-between text-sm">
                    <span>
                      #{run.ID} · {formatNextRun(run.CreatedAt)}
                    </span>
                    <Badge variant="outline">{run.Status}</Badge>
                  </li>
                ))}
              </ul>
            </div>
            <DialogFooter className="flex justify-between">
              <Button
                variant="destructive"
                onClick={() => {
                  if (confirm(ui.procurement.scheduleDeleteConfirm)) {
                    deleteMutation.mutate(detail.Schedule.ID, {
                      onSuccess: onClose,
                    });
                  }
                }}
              >
                {ui.procurement.scheduleDelete}
              </Button>
              <Button variant="outline" onClick={onClose}>
                {ui.common.cancel}
              </Button>
            </DialogFooter>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}


function ScheduleFormDialog({
  schedule,
  suppliers,
  products,
  onClose,
}: {
  schedule: ScheduleDto | null;
  suppliers: SupplierDto[];
  products: ProductDto[];
  onClose: () => void;
}) {
  const createMutation = useCreateSchedule();
  const updateMutation = useUpdateSchedule();
  const [selectedDays, setSelectedDays] = useState<number[]>(
    schedule?.DaysOfWeek ?? [1, 2, 3, 4, 5]
  );
  const [selectedProducts, setSelectedProducts] = useState<number[]>(
    schedule?.ProductIDs ?? []
  );
  const [selectedSuppliers, setSelectedSuppliers] = useState<number[]>(
    schedule?.SupplierIDs ?? []
  );

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<ScheduleFormValues>({
    defaultValues: {
      name: schedule?.Name ?? "",
      run_time: schedule?.RunTime ?? "08:00",
      note: schedule?.Note ?? "",
    },
  });

  const toggle = (list: number[], value: number, setList: (v: number[]) => void) => {
    setList(list.includes(value) ? list.filter((v) => v !== value) : [...list, value]);
  };

  const onSubmit = handleSubmit(async (values) => {
    const payload = {
      name: values.name.trim(),
      run_time: values.run_time,
      days_of_week: selectedDays,
      product_ids: selectedProducts,
      supplier_ids: selectedSuppliers,
      note: values.note?.trim() || null,
    };
    if (!payload.name || !payload.run_time) {
      toast.error("El nombre y la hora de ejecución son obligatorios");
      return;
    }
    if (payload.days_of_week.length === 0) {
      toast.error("Debe seleccionar al menos un día de la semana");
      return;
    }
    if (payload.product_ids.length === 0 || payload.supplier_ids.length === 0) {
      toast.error("Debe seleccionar al menos un producto y un proveedor");
      return;
    }
    try {
      if (schedule) {
        await updateMutation.mutateAsync({ id: schedule.ID, data: payload });
      } else {
        await createMutation.mutateAsync(payload);
      }
      toast.success(ui.procurement.scheduleSaved);
      onClose();
    } catch {
      // mutation onError already toasts the Spanish error
    }
  });

  return (
    <Dialog open onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {schedule ? ui.procurement.editSchedule : ui.procurement.addSchedule}
          </DialogTitle>
        </DialogHeader>
        <form onSubmit={onSubmit} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="schedule-name">{ui.procurement.scheduleName}</Label>
            <Input
              id="schedule-name"
              placeholder={ui.procurement.scheduleNamePlaceholder}
              {...register("name")}
            />
            {errors.name && <p className="text-sm text-red-600">{errors.name.message}</p>}
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="schedule-run-time">{ui.procurement.runTime}</Label>
            <Input
              id="schedule-run-time"
              type="time"
              {...register("run_time", {
                required: ui.procurement.runTime,
                pattern: {
                  value: /^([01]\d|2[0-3]):[0-5]\d$/,
                  message: "HH:MM",
                },
              })}
            />
            {errors.run_time && <p className="text-sm text-red-600">{errors.run_time.message}</p>}
          </div>
          <div className="space-y-1.5">
            <Label>{ui.procurement.daysOfWeek}</Label>
            <div className="flex flex-wrap gap-2">
              {WEEKDAYS.map((d) => (
                <button
                  key={d.value}
                  type="button"
                  onClick={() => toggle(selectedDays, d.value, setSelectedDays)}
                  className={`rounded-md border px-3 py-1 text-sm ${
                    selectedDays.includes(d.value)
                      ? "border-blue-600 bg-blue-50 text-blue-700"
                      : "border-gray-300 text-gray-600 hover:bg-gray-50"
                  }`}
                >
                  {d.label.slice(0, 3)}
                </button>
              ))}
            </div>
          </div>
          <div className="space-y-1.5">
            <Label>{ui.procurement.productsIncluded}</Label>
            <div className="max-h-32 overflow-y-auto rounded-md border p-2">
              {products.map((p) => (
                <label key={p.id} className="flex items-center gap-2 py-0.5 text-sm">
                  <input
                    type="checkbox"
                    checked={selectedProducts.includes(p.id)}
                    onChange={() => toggle(selectedProducts, p.id, setSelectedProducts)}
                  />
                  {p.name} ({p.sku})
                </label>
              ))}
              {products.length === 0 && (
                <p className="text-sm text-gray-400">{ui.procurement.productsEmpty}</p>
              )}
            </div>
          </div>
          <div className="space-y-1.5">
            <Label>{ui.procurement.suppliersIncluded}</Label>
            <div className="max-h-32 overflow-y-auto rounded-md border p-2">
              {suppliers.map((s) => (
                <label key={s.id} className="flex items-center gap-2 py-0.5 text-sm">
                  <input
                    type="checkbox"
                    checked={selectedSuppliers.includes(s.id)}
                    onChange={() => toggle(selectedSuppliers, s.id, setSelectedSuppliers)}
                  />
                  {s.display_name || s.nit}
                </label>
              ))}
              {suppliers.length === 0 && (
                <p className="text-sm text-gray-400">{ui.procurement.suppliersEmpty}</p>
              )}
            </div>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="schedule-note">{ui.procurement.notes}</Label>
            <Input id="schedule-note" {...register("note")} />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              {ui.common.cancel}
            </Button>
            <Button type="submit" disabled={createMutation.isPending || updateMutation.isPending}>
              {ui.common.save}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
