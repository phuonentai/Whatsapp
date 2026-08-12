/**
 * RBAC Roles Query Hook
 *
 * Fetches the role/permission definitions from `GET /api/rbac/roles` (served
 * from the Stytch RBAC policy — the runtime SSOT).
 *
 * Freshness contract: `staleTime` MUST NOT exceed the Stytch policy cache TTL
 * (5 minutes) so the matrix and the role selector never show a policy older
 * than the backend cache; `refetchOnWindowFocus` plus an explicit manual
 * refetch control keep the data current without hammering the API.
 */

import { useQuery, type UseQueryOptions } from "@tanstack/react-query";
import {
  rbacRepository,
  type RbacRole,
} from "@/lib/api/api/repositories/rbac-repository";
import { queryKeys } from "./query-keys";

/** Policy cache TTL on the backend (Stytch policy cache, Redis). */
export const RBAC_POLICY_TTL_MS = 5 * 60 * 1000;

interface UseRbacRolesQueryOptions {
  enabled?: boolean;
}

export function useRbacRolesQuery(
  options: UseRbacRolesQueryOptions = {},
  queryOptions?: Omit<
    UseQueryOptions<RbacRole[], Error>,
    "queryKey" | "queryFn" | "enabled"
  >
) {
  const { enabled = true } = options;

  return useQuery({
    queryKey: queryKeys.rbac.roles(),
    // forceRefresh bypasses the repository's own in-memory cache so the
    // TanStack Query cache owns freshness (staleTime ≤ policy TTL).
    queryFn: () => rbacRepository.getRoles(true),

    enabled,

    // Same TTL as the Stytch policy cache (contract: staleTime ≤ 5 min).
    staleTime: RBAC_POLICY_TTL_MS,

    // Cache for 10 minutes
    gcTime: 10 * 60 * 1000,

    // Keep data when the policy is temporarily unavailable.
    retry: 1,

    // Refetch when the user returns to the tab so the matrix stays current.
    refetchOnWindowFocus: true,

    ...queryOptions,
  });
}
