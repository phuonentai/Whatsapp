"use client";

import { useState } from "react";
import { useTicketsQuery } from "@/lib/hooks/queries/use-modules-queries";
import { useCreateTicket } from "@/lib/hooks/mutations/use-tickets-mutations";
import { ErrorState } from "@/components/common/error-state";
import type { TicketDto } from "@/lib/api/api/repositories/ticket-repository";

const STATUSES = [
  { value: "", label: "Todos" },
  { value: "open", label: "Abiertos" },
  { value: "in_progress", label: "En progreso" },
  { value: "waiting_customer", label: "Esperando cliente" },
  { value: "resolved", label: "Resueltos" },
] as const;

const PRIORITY_LABELS: Record<TicketDto["priority"], string> = {
  low: "Baja",
  normal: "Normal",
  high: "Alta",
};

export function TicketList({
  selectedId,
  onSelect,
  statusFilter,
  onStatusFilterChange,
}: {
  selectedId: number | null;
  onSelect: (id: number) => void;
  statusFilter: string;
  onStatusFilterChange: (status: string) => void;
}) {
  const { data: tickets, isLoading, isError, refetch, isRefetching } = useTicketsQuery(statusFilter ? { status: statusFilter } : undefined);
  const createTicket = useCreateTicket();
  const [creating, setCreating] = useState(false);
  const [title, setTitle] = useState("");

  const handleCreate = () => {
    if (!title.trim()) return;
    createTicket.mutate({ title: title.trim() });
    setTitle("");
    setCreating(false);
  };

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <select
          value={statusFilter}
          onChange={(e) => onStatusFilterChange(e.target.value)}
          className="border rounded px-2 py-1 text-sm"
        >
          {STATUSES.map((s) => (
            <option key={s.value} value={s.value}>
              {s.label}
            </option>
          ))}
        </select>
        <button
          onClick={() => setCreating(!creating)}
          className="px-3 py-1 bg-blue-600 text-white rounded text-sm"
        >
          Nuevo ticket
        </button>
      </div>

      {creating && (
        <div className="flex gap-2">
          <input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="Título del ticket"
            className="border rounded px-2 py-1 text-sm flex-1"
          />
          <button onClick={handleCreate} className="px-3 py-1 bg-green-600 text-white rounded text-sm">
            Crear
          </button>
        </div>
      )}

      {isLoading && <div className="text-gray-500 text-sm">Cargando tickets...</div>}

      {isError && (
        <ErrorState
          title="Error al cargar los tickets"
          description="No se pudieron cargar los tickets. Inténtalo de nuevo."
          onRetry={() => refetch()}
          isRetrying={isRefetching}
        />
      )}

      <div className="space-y-2">
        {(tickets ?? []).map((t) => (
          <button
            key={t.id}
            onClick={() => onSelect(t.id)}
            className={`w-full text-left border rounded-lg p-3 bg-white hover:bg-gray-50 ${
              selectedId === t.id ? "ring-2 ring-blue-500" : ""
            }`}
          >
            <div className="flex items-center justify-between">
              <span className="font-medium text-sm truncate">{t.title}</span>
              <span className="text-xs text-gray-500">#{t.id}</span>
            </div>
            <div className="flex items-center gap-2 mt-1 text-xs">
              <span
                className={`px-1.5 py-0.5 rounded ${
                  t.priority === "high"
                    ? "bg-red-100 text-red-700"
                    : t.priority === "normal"
                    ? "bg-yellow-100 text-yellow-700"
                    : "bg-gray-100 text-gray-600"
                }`}
              >
                {PRIORITY_LABELS[t.priority]}
              </span>
              <span className="text-gray-500">{t.status.replace("_", " ")}</span>
              {t.overdue && <span className="text-red-600 font-medium">SLA vencido</span>}
            </div>
          </button>
        ))}
        {tickets && tickets.length === 0 && (
          <div className="text-gray-500 text-sm p-4 border rounded bg-white">Sin tickets.</div>
        )}
      </div>
    </div>
  );
}
