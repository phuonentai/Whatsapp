/**
 * Remove Member Mutation
 *
 * Handles member removal with optimistic UI updates.
 * Immediately removes member from UI, then rolls back on error.
 */

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { memberRepository } from "@/lib/api/api/repositories/member-repository";
import { queryKeys } from "../queries/query-keys";
import type { MemberListResponse } from "@/lib/models/member.model";

interface RemoveMemberVariables {
  memberId: string;
}

export function useRemoveMember() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ memberId }: RemoveMemberVariables) =>
      memberRepository.removeMember(memberId),

    // Optimistic update: Remove member from UI immediately
    onMutate: async ({ memberId }) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({
        queryKey: queryKeys.members.all,
      });

      // Snapshot all members queries for this organization
      const previousQueries = queryClient.getQueriesData<MemberListResponse>({
        queryKey: queryKeys.members.all,
      });

      // Optimistically remove the member from all cached queries
      queryClient.setQueriesData<MemberListResponse>(
        { queryKey: queryKeys.members.all },
        (old) => {
          if (!old) return old;

          return {
            ...old,
            members: old.members.filter((member) => member.id !== memberId),
            totalCount: old.totalCount - 1,
          };
        }
      );

      // Return context for rollback
      return { previousQueries };
    },

    // On success, log it
    onSuccess: (_data, variables) => {
      console.info("[Mutation] Member removed successfully:", {
        memberId: variables.memberId,
      });
    },

    // On error, rollback to previous state
    onError: (error, variables, context) => {
      console.error("[Mutation] Member removal failed:", {
        error,
        memberId: variables.memberId,
      });

      // Restore all previous queries
      if (context?.previousQueries) {
        context.previousQueries.forEach(([queryKey, data]) => {
          queryClient.setQueryData(queryKey, data);
        });
      }
    },

    // Always refetch to ensure consistency
    onSettled: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.members.all,
      });
    },
  });
}
