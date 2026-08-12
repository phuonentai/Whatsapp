import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { moduleRepository } from "@/lib/api/api/repositories/module-repository";
import { ticketRepository, type TicketDto } from "@/lib/api/api/repositories/ticket-repository";
import { copy } from "@/lib/copy/ui";
import { queryKeys } from "../queries/query-keys";

export function useSaveModuleConfig() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ key, config }: { key: string; config: Record<string, unknown> }) =>
      moduleRepository.saveConfig(key, config),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.modules.org() });
      queryClient.invalidateQueries({ queryKey: queryKeys.crm.entitlement() });
      toast.success("Configuración guardada");
    },
    onError: () => toast.error("No se pudo guardar la configuración"),
  });
}

export function useCreateTicket() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ticketRepository.create,
    onSuccess: (ticket) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.crm.tickets() });
      queryClient.invalidateQueries({ queryKey: queryKeys.crm.ticket(ticket.id) });
      toast.success("Ticket creado");
    },
    onError: () => toast.error("No se pudo crear el ticket"),
  });
}

export function useTransitionTicket() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, status }: { id: number; status: TicketDto["status"] }) =>
      ticketRepository.transition(id, status),
    onSuccess: (ticket) => {
      invalidateTicket(queryClient, ticket);
    },
    onError: () => toast.error("No se pudo cambiar el estado"),
  });
}

export function useAssignTicket() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, assignee }: { id: number; assignee: string }) =>
      ticketRepository.assign(id, assignee),
    onSuccess: (ticket) => invalidateTicket(queryClient, ticket),
    onError: () => toast.error("No se pudo asignar el ticket"),
  });
}

export function useSetTicketPriority() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, priority }: { id: number; priority: TicketDto["priority"] }) =>
      ticketRepository.setPriority(id, priority),
    onSuccess: (ticket) => invalidateTicket(queryClient, ticket),
    onError: () => toast.error("No se pudo cambiar la prioridad"),
  });
}

export function useAddInternalNote() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, body }: { id: number; body: string }) => ticketRepository.addInternalNote(id, body),
    onSuccess: (_event, vars) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.crm.ticket(vars.id) });
      toast.success("Nota interna agregada");
    },
    onError: () => toast.error("No se pudo agregar la nota"),
  });
}

// useAiTriageMutation drafts an internal note and suggests a priority for a
// ticket. Triage never mutates the ticket, so nothing is invalidated; the
// caller fills the draft and shows the suggestion. Failure (including 402
// ai_credits_exhausted) surfaces as a toast and leaves the form untouched.
export function useAiTriageMutation() {
  return useMutation({
    mutationFn: ({ id }: { id: number }) => ticketRepository.aiTriage(id),
    onError: (error) => {
      if (error instanceof Error && error.message.includes("ai_credits_exhausted")) {
        toast.error(copy("tickets", "triageCreditsExhausted"));
        return;
      }
      toast.error(copy("tickets", "triageError"));
    },
  });
}

function invalidateTicket(queryClient: ReturnType<typeof useQueryClient>, ticket: TicketDto) {
  queryClient.invalidateQueries({ queryKey: queryKeys.crm.tickets() });
  queryClient.invalidateQueries({ queryKey: queryKeys.crm.ticket(ticket.id) });
}
