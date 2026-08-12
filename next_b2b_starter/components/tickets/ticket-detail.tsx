"use client";

import { useState } from "react";
import { useTicketQuery } from "@/lib/hooks/queries/use-modules-queries";
import {
  useTransitionTicket,
  useSetTicketPriority,
  useAddInternalNote,
  useAiTriageMutation,
} from "@/lib/hooks/mutations/use-tickets-mutations";
import { ui, tpl } from "@/lib/copy/ui";
import { ErrorState } from "@/components/common/error-state";
import type { TicketDto } from "@/lib/api/api/repositories/ticket-repository";

const STATUS_OPTIONS: { value: TicketDto["status"]; label: string }[] = [
  { value: "open", label: "Abierto" },
  { value: "in_progress", label: "En progreso" },
  { value: "waiting_customer", label: "Esperando cliente" },
  { value: "resolved", label: "Resuelto" },
  { value: "cancelled", label: "Cancelado" },
];

const PRIORITY_LABELS: Record<TicketDto["priority"], string> = {
  low: "baja",
  normal: "media",
  high: "alta",
};

export function TicketDetail({ id }: { id: number }) {
  const { data, isLoading, isError, refetch, isRefetching } = useTicketQuery(id);
  const transition = useTransitionTicket();
  const setPriority = useSetTicketPriority();
  const addNote = useAddInternalNote();
  const triage = useAiTriageMutation();
  const [note, setNote] = useState("");
  const [prioritySuggestion, setPrioritySuggestion] = useState<TicketDto["priority"] | null>(null);

  if (isLoading) return <div className="text-gray-500 text-sm">Cargando...</div>;

  if (isError || !data) {
    return (
      <ErrorState
        title="Error al cargar el ticket"
        description="No se pudo cargar el ticket. Inténtalo de nuevo."
        onRetry={() => refetch()}
        isRetrying={isRefetching}
      />
    );
  }

  const { ticket, eventos } = data;

  return (
    <div className="border rounded-lg bg-white p-4 space-y-4">
      <div>
        <div className="flex items-center justify-between">
          <h3 className="font-semibold">#{ticket.id} · {ticket.title}</h3>
          {ticket.overdue && <span className="text-red-600 text-xs font-medium">SLA vencido</span>}
        </div>
        {ticket.description && <p className="text-sm text-gray-600 mt-1">{ticket.description}</p>}
      </div>

      <div className="flex flex-wrap items-center gap-2 text-sm">
        <select
          value={ticket.status}
          onChange={(e) => transition.mutate({ id: ticket.id, status: e.target.value as TicketDto["status"] })}
          className="border rounded px-2 py-1 text-sm"
        >
          {STATUS_OPTIONS.map((s) => (
            <option key={s.value} value={s.value}>
              {s.label}
            </option>
          ))}
        </select>
        <select
          value={ticket.priority}
          onChange={(e) => setPriority.mutate({ id: ticket.id, priority: e.target.value as TicketDto["priority"] })}
          className="border rounded px-2 py-1 text-sm"
        >
          <option value="low">Prioridad baja</option>
          <option value="normal">Prioridad normal</option>
          <option value="high">Prioridad alta</option>
        </select>
        {ticket.sla_due_at && (
          <span className="text-xs text-gray-500">
            SLA: {new Date(ticket.sla_due_at).toLocaleString()}
          </span>
        )}
      </div>

      <div>
        <h4 className="text-xs font-semibold text-gray-500 uppercase mb-2">Historial</h4>
        <div className="space-y-1.5 max-h-72 overflow-y-auto">
          {(eventos ?? []).map((e) => (
            <div key={e.id} className="text-xs text-gray-600 border-b pb-1.5">
              <span className="text-gray-400">{new Date(e.created_at).toLocaleString()}</span>{" "}
              <span className="font-medium">{e.event_type.replace(/_/g, " ")}</span>
              {e.event_type === "note_internal" && typeof e.payload?.body === "string" && (
                <span className="block mt-0.5 text-gray-700">💬 {e.payload.body}</span>
              )}
            </div>
          ))}
          {!eventos?.length && <div className="text-gray-400">Sin eventos.</div>}
        </div>
      </div>

      {prioritySuggestion && (
        <div className="flex items-center gap-2 text-xs bg-amber-50 border border-amber-200 rounded px-2 py-1.5">
          <span className="text-amber-800">
            {tpl(ui.tickets.triagePrioritySuggestion, { priority: PRIORITY_LABELS[prioritySuggestion] })}
          </span>
          <button
            onClick={() => {
              setPriority.mutate({ id: ticket.id, priority: prioritySuggestion });
              setPrioritySuggestion(null);
            }}
            className="px-2 py-0.5 bg-amber-600 text-white rounded text-xs"
          >
            {ui.tickets.triageApply}
          </button>
          <button
            onClick={() => setPrioritySuggestion(null)}
            aria-label="Descartar sugerencia de prioridad"
            className="ml-auto text-amber-700 hover:text-amber-900"
          >
            ×
          </button>
        </div>
      )}

      <div className="flex gap-2">
        <input
          value={note}
          onChange={(e) => setNote(e.target.value)}
          placeholder="Nota interna (solo equipo)..."
          className="border rounded px-2 py-1 text-sm flex-1"
        />
        <button
          onClick={() => {
            if (!note.trim()) return;
            addNote.mutate({ id: ticket.id, body: note.trim() });
            setNote("");
          }}
          className="px-3 py-1 bg-gray-800 text-white rounded text-sm"
        >
          Agregar
        </button>
        <button
          onClick={() =>
            triage.mutate(
              { id: ticket.id },
              {
                onSuccess: (result) => {
                  setNote(result.note);
                  setPrioritySuggestion(result.priority);
                },
              }
            )
          }
          disabled={triage.isPending}
          className="px-3 py-1 bg-emerald-500 text-white rounded text-sm disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {triage.isPending ? "Generando…" : ui.tickets.triageDraft}
        </button>
      </div>
    </div>
  );
}
