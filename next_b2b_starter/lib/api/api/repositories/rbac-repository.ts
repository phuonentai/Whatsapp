// lib/api/api/repositories/rbac-repository.ts

import { apiClient } from "../client/api-client";
import type {
  RbacRolesResponseDto,
  RbacRoleDto,
  RbacPermissionDto,
} from "../dto/rbac.dto";

export interface RbacPermission {
  id: string;
  resource: string;
  action: string;
  displayName: string;
  description: string;
  category: string;
}

export interface RbacRole {
  id: string;
  name: string;
  description: string;
  typicalUsers: string;
  permissions: RbacPermission[];
}

class RbacRepository {
  private cachedRoles: RbacRole[] | null = null;

  /**
   * Fetch all RBAC roles from backend.
   * Response is cached in-memory since role definitions are static.
   */
  async getRoles(forceRefresh = false): Promise<RbacRole[]> {
    if (!forceRefresh && this.cachedRoles) {
      return this.cachedRoles;
    }

    const response = await apiClient.get<RbacRolesResponseDto>("/rbac/roles", {
      skipAuth: true,
    });

    const roles = (response.roles ?? []).map((role) => this.toRbacRole(role));
    this.cachedRoles = roles;
    return roles;
  }

  private toRbacRole(dto: RbacRoleDto): RbacRole {
    return {
      id: dto.id,
      name: dto.name,
      description: dto.description,
      typicalUsers: dto.typical_users,
      permissions: (dto.permissions ?? []).map((permission) =>
        this.toRbacPermission(permission)
      ),
    };
  }

  private toRbacPermission(dto: RbacPermissionDto): RbacPermission {
    return {
      id: dto.id,
      resource: dto.resource,
      action: dto.action,
      displayName: dto.display_name,
      description: dto.description,
      category: dto.category,
    };
  }
}

export const rbacRepository = new RbacRepository();
