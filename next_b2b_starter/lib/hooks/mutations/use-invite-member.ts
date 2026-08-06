/**
 * Invite Member Mutation
 *
 * Handles member invitation with automatic members list invalidation.
 */

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { memberRepository } from "@/lib/api/api/repositories/member-repository";
import { queryKeys } from "../queries/query-keys";
import type {
  InviteMemberRequest,
  InviteMemberResponse,
} from "@/lib/models/member.model";

interface InviteMemberVariables {
  request: InviteMemberRequest;
  organizationId: string;
}

export function useInviteMember() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ request, organizationId }: InviteMemberVariables) =>
      memberRepository.inviteMember(request, organizationId),

    // On success, invalidate members list to refetch with new member
    onSuccess: (data: InviteMemberResponse, variables) => {
      console.info("[Mutation] Member invited successfully:", {
        memberId: data.memberId,
        organizationId: variables.organizationId,
      });

      // Invalidate all member lists for this organization
      queryClient.invalidateQueries({
        queryKey: queryKeys.members.all,
      });
    },

    // On error, log the failure
    onError: (error, variables) => {
      console.error("[Mutation] Member invitation failed:", {
        error,
        email: variables.request.email,
        organizationId: variables.organizationId,
      });
    },
  });
}
