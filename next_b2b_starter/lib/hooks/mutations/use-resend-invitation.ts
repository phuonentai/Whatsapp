/**
 * Resend Invitation Mutation
 *
 * Handles resending invitation emails to pending members.
 * No optimistic updates needed since UI doesn't change.
 */

import { useMutation } from "@tanstack/react-query";
import { memberRepository } from "@/lib/api/api/repositories/member-repository";

interface ResendInvitationVariables {
  memberId: string;
}

export function useResendInvitation() {
  return useMutation({
    mutationFn: ({ memberId }: ResendInvitationVariables) =>
      memberRepository.resendInvitation(memberId),

    // On success, log it
    onSuccess: (_data, variables) => {
      console.info("[Mutation] Invitation resent successfully:", {
        memberId: variables.memberId,
      });
    },

    // On error, log the failure
    onError: (error, variables) => {
      console.error("[Mutation] Resend invitation failed:", {
        error,
        memberId: variables.memberId,
      });
    },
  });
}
