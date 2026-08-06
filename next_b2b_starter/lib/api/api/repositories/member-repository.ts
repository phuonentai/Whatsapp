// lib/api/api/repositories/member-repository.ts

import { apiClient } from "../client/api-client";
import {
  MemberListResponseDto,
  InviteMemberRequestDto,
  InviteMemberResponseDto,
  RemoveMemberRequestDto,
  RemoveMemberResponseDto,
  MemberDto,
  UpdateProfileRequestDto,
  UpdateProfileResponseDto,
  ResendInvitationRequestDto,
  ResendInvitationResponseDto,
} from "../dto/member.dto";
import { ProfileResponseDto } from "../dto/profile.dto";
import {
  OrganizationMember,
  UserProfile,
  InviteMemberRequest,
  InviteMemberResponse,
  UpdateProfileRequest,
  MemberListResponse,
  MemberRole,
} from "@/lib/models/member.model";

class MemberRepository {
  /**
   * Get current user profile
   */
  async getProfile(): Promise<UserProfile> {
    type ProfileApiResponse =
      | ProfileResponseDto
      | {
          data?: ProfileResponseDto;
          profile?: ProfileResponseDto;
          success?: boolean;
          message?: string;
        };

    const response = await apiClient.get<ProfileApiResponse>("/auth/profile/me");

    const profileDto =
      (response as { data?: ProfileResponseDto }).data ??
      (response as { profile?: ProfileResponseDto }).profile ??
      (response as ProfileResponseDto);

    if (!profileDto || !profileDto.member_id) {
      const errorMessage =
        (response as { message?: string }).message ||
        "Profile response did not include member information";
      throw new Error(errorMessage);
    }

    return this.toUserProfile(profileDto);
  }

  /**
   * Update user profile
   */
  async updateProfile(request: UpdateProfileRequest): Promise<UserProfile> {
    const payload: UpdateProfileRequestDto = {
      name: request.name,
      avatar_url: request.avatarUrl,
    };

    type UpdateProfileApiResponse = UpdateProfileResponseDto & {
      profile?: ProfileResponseDto;
      data?: ProfileResponseDto;
    };

    const response = await apiClient.put<UpdateProfileApiResponse>(
      "/auth/profile/me",
      payload
    );

    if (response.profile || response.data) {
      return this.toUserProfile(response.profile ?? response.data!);
    }

    if (!response.success) {
      throw new Error(response.message || "Failed to update profile");
    }

    // Fetch updated profile after successful update
    return this.getProfile();
  }

  /**
   * Get organization members list
   * You can optionally scope results by organization and pagination settings.
   */
  async getMembers(options?: {
    organizationId?: string;
    page?: number;
    pageSize?: number;
  }): Promise<MemberListResponse> {
    const params = new URLSearchParams();

    if (options?.organizationId) {
      params.append("organization_id", options.organizationId);
    }

    if (options?.page) {
      params.append("page", String(options.page));
    }

    if (options?.pageSize) {
      params.append("page_size", String(options.pageSize));
    }

    const queryString = params.toString();
    const endpoint = queryString ? `/auth/members?${queryString}` : "/auth/members";

    type MemberListApiResponse =
      | MemberListResponseDto
      | {
          data?: MemberListResponseDto;
          members?: MemberDto[];
          total?: number;
          success?: boolean;
        };

    const response = await apiClient.get<MemberListApiResponse>(endpoint);

    const dto: MemberListResponseDto = "data" in response && response.data
      ? response.data
      : {
          members:
            (response as { members?: MemberDto[] }).members ??
            (response as MemberListResponseDto).members,
          total:
            (response as { total?: number }).total ??
            (response as MemberListResponseDto).total,
        };

    return {
      members: (dto.members ?? []).map((item) => this.toOrganizationMember(item)),
      totalCount: dto.total ?? dto.members?.length ?? 0,
      hasMore: false, // Backend doesn't provide this, calculate if needed
    };
  }

  /**
   * Invite a new member
   */
  async inviteMember(
    request: InviteMemberRequest,
    organizationId: string
  ): Promise<InviteMemberResponse> {
    const payload: InviteMemberRequestDto = {
      email: request.email,
      name: request.name,
      role_slug: request.role,
    };

    const response = await apiClient.post<InviteMemberResponseDto>(
      "/auth/members",
      payload
    );

    return {
      success: response.success,
      memberId: response.member_id,
      message: response.message,
      inviteLink: response.invite_link,
    };
  }

  /**
   * Remove member from organization
   */
  async removeMember(memberId: string): Promise<boolean> {
    const response = await apiClient.delete<RemoveMemberResponseDto>(
      `/auth/members/${memberId}`
    );

    return response.success;
  }

  /**
   * Resend invitation to pending member
   */
  async resendInvitation(memberId: string): Promise<boolean> {
    const response = await apiClient.post<ResendInvitationResponseDto>(
      `/members/${memberId}/resend-invitation`,
      { member_id: memberId }
    );

    return response.success;
  }

  /**
   * Transform DTO to UserProfile model
   * Extracts first non-Stytch role from roles array as the primary role
   */
  private toUserProfile(dto: ProfileResponseDto): UserProfile {
    // Log received DTO for debugging
    console.log("[MemberRepository] Profile DTO received:", {
      member_id: dto.member_id,
      email: dto.email,
      name: dto.name,
      roles: dto.roles,
      organization: dto.organization,
    });

    // Extract first non-stytch role as the primary role with null safety
    const roles = dto.roles || [];
    const primaryRole =
      roles.find((r) => !r.startsWith("stytch_")) || roles[0] || "member";
    const normalizedRole: MemberRole = ["admin", "manager", "member"].includes(
      primaryRole
    )
      ? (primaryRole as MemberRole)
      : "member";

    return {
      id: dto.member_id,
      email: dto.email,
      name: dto.name,
      avatarUrl: undefined, // Not in backend response
      role: normalizedRole,
      organizationId: dto.organization?.organization_id || "",
      organizationName: dto.organization?.name || "",
    };
  }

  /**
   * Transform DTO to OrganizationMember model
   * Extracts first non-Stytch role from roles array as the primary role
   */
  private toOrganizationMember(dto: MemberDto): OrganizationMember {
    // Extract first non-stytch role as the primary role with null safety
    const roles = dto.roles || [];
    const primaryRole =
      roles.find((r) => !r.startsWith("stytch_")) || roles[0] || "member";
    const normalizedRole: MemberRole = ["admin", "manager", "member"].includes(
      primaryRole
    )
      ? (primaryRole as MemberRole)
      : "member";

    return {
      id: dto.member_id,
      email: dto.email,
      name: dto.name,
      role: normalizedRole,
      status: dto.status as "active" | "pending" | "inactive",
      avatarUrl: undefined, // Not in backend response
      joinedAt: new Date(dto.created_at),
      invitedAt: undefined, // Not in backend response
      invitedBy: undefined, // Not in backend response
    };
  }
}

export const memberRepository = new MemberRepository();
