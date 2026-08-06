// lib/api/api/dto/rbac.dto.ts

export interface RbacPermissionDto {
  id: string;
  resource: string;
  action: string;
  display_name: string;
  description: string;
  category: string;
}

export interface RbacRoleDto {
  id: string;
  name: string;
  description: string;
  typical_users: string;
  permissions: RbacPermissionDto[];
}

export interface RbacRolesResponseDto {
  roles: RbacRoleDto[];
}
