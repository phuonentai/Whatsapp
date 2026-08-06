export interface CreateOrganizationRequestDto {
  name: string;
  industry: string;
}

export interface CreateOrganizationResponseDto {
  organization_id: number;
  name: string;
  industry: string;
  created_at: string;
  updated_at: string;
}
