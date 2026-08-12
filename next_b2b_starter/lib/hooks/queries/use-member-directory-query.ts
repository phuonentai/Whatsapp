import { useQuery } from "@tanstack/react-query";
import { conversationRepository } from "@/lib/api/api/repositories/conversation-repository";
import { queryKeys } from "./query-keys";

/**
 * Directorio de miembros activos del org para el picker de re-asignación
 * (conversation-row-scoping). Cuando el endpoint responde 503
 * (member_directory_unavailable — circuit abierto / cache vacía), el query
 * queda en estado de error y el picker muestra estado de retry (sin ghost);
 * el thread y el composer permanecen operativos.
 */
export function useMemberDirectoryQuery() {
  return useQuery({
    queryKey: queryKeys.crm.memberDirectory(),
    queryFn: () => conversationRepository.listMemberDirectory(),
    staleTime: 5 * 60 * 1000,
    retry: 1,
  });
}
