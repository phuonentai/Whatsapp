/**
 * Profile Repository - Fetch current user profile with computed permissions
 */

import { apiClient } from "../client/api-client";
import type { ProfileResponseDto } from "../dto/profile.dto";

class ProfileRepository {
  /**
   * Get current user profile with backend-computed permissions
   * Backend resolves Stytch RBAC policy and returns expanded permissions
   *
   * @param sessionToken - Optional JWT token. If provided, uses this token directly.
   *                      If not provided, API client will read JWT from cookies automatically.
   *                      Server-side calls should pass the token explicitly.
   *                      Client-side calls can omit it to use automatic cookie-based auth.
   * @returns Profile with computed permissions array
   */
  async getProfile(sessionToken?: string): Promise<ProfileResponseDto> {
    const options = sessionToken
      ? {
          headers: {
            Authorization: `Bearer ${sessionToken}`,
          },
        }
      : undefined;

    type ProfileApiResponse =
      | ProfileResponseDto
      | {
          data?: ProfileResponseDto;
          success?: boolean;
          message?: string;
        };

    const response = await apiClient.get<ProfileApiResponse>(
      "/auth/profile/me",
      options
    );

    // Backend wraps response in { data: {...}, success: true }
    // Extract the actual profile data
    const profileDto =
      (response as { data?: ProfileResponseDto }).data ??
      (response as ProfileResponseDto);

    if (!profileDto || !profileDto.member_id) {
      const errorMessage =
        (response as { message?: string }).message ||
        "Profile response did not include member information";
      throw new Error(errorMessage);
    }

    return profileDto;
  }
}

export const profileRepository = new ProfileRepository();
