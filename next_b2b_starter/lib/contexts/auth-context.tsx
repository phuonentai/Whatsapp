'use client';

import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

import type { ProfileResponseDto } from "@/lib/api/api/dto/profile.dto";
import {
  hasPermission as checkPermission,
  hasAnyPermission as checkAnyPermission,
  hasAllPermissions as checkAllPermissions,
  hasRole as checkRole,
  hasAnyRole as checkAnyRole,
  hasAllRoles as checkAllRoles,
} from "@/lib/auth/permission-utils";
import { isTokenExpired } from "@/lib/auth/token-utils";
import { SESSION_JWT_COOKIE_NAME } from "@/lib/auth/constants";

const STORAGE_KEY = "apcash.auth.state";
const SESSION_DURATION_MS = 8 * 60 * 60 * 1000; // 8 hours in milliseconds

type AuthState = {
  profile: ProfileResponseDto | null;
  roles: string[];
  permissions: string[];
  expiresAt?: number; // Unix timestamp in milliseconds
};

function sanitizeRoles(roles: unknown): string[] {
  if (!Array.isArray(roles)) {
    return [];
  }

  return roles
    .filter((role): role is string => typeof role === "string")
    .map((role) => role.trim())
    .filter(Boolean);
}

function sanitizePermissions(permissions: unknown): string[] {
  if (!Array.isArray(permissions)) {
    return [];
  }

  // Trust backend as source of truth - only validate type and filter Stytch internal permissions
  return permissions.filter(
    (permission): permission is string =>
      typeof permission === "string" &&
      permission.trim().length > 0 &&
      !permission.startsWith('stytch.')
  );
}

export interface AuthContextValue {
  profile: ProfileResponseDto | null;
  roles: string[];
  permissions: string[];
  hasPermission: (permission: string) => boolean;
  hasAnyPermission: (permissions: string[]) => boolean;
  hasAllPermissions: (permissions: string[]) => boolean;
  hasRole: (role: string) => boolean;
  hasAnyRole: (roles: string[]) => boolean;
  hasAllRoles: (roles: string[]) => boolean;
  isInitialized: boolean;
  isAuthenticated: boolean;
  updateAuthState: (state: Partial<AuthState>) => void;
  clearAuthState: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

function readStoredState(): AuthState | null {
  if (typeof window === "undefined") {
    return null;
  }

  try {
    const raw = window.sessionStorage.getItem(STORAGE_KEY);
    if (!raw) {
      return null;
    }

    const parsed = JSON.parse(raw) as AuthState;
    if (!parsed || typeof parsed !== "object") {
      return null;
    }

    // Check if cached data has expired
    if (parsed.expiresAt && Date.now() > parsed.expiresAt) {
      console.info("[Auth] Cached auth state expired, clearing");
      window.sessionStorage.removeItem(STORAGE_KEY);
      return null;
    }

    return {
      profile: parsed.profile ?? null,
      roles: sanitizeRoles(parsed.roles),
      permissions: sanitizePermissions(parsed.permissions),
      expiresAt: parsed.expiresAt,
    };
  } catch (error) {
    console.warn("[Auth] Failed to parse stored auth state", error);
    return null;
  }
}

function persistState(state: AuthState): void {
  if (typeof window === "undefined") {
    return;
  }

  try {
    if (!state.profile) {
      window.sessionStorage.removeItem(STORAGE_KEY);
      return;
    }

    // Calculate expiration time (8 hours from now)
    const expiresAt = Date.now() + SESSION_DURATION_MS;

    const payload: AuthState = {
      profile: state.profile,
      roles: state.roles,
      permissions: state.permissions,
      expiresAt,
    };

    window.sessionStorage.setItem(STORAGE_KEY, JSON.stringify(payload));
  } catch (error) {
    console.warn("[Auth] Failed to persist auth state", error);
  }
}

function readBrowserCookie(name: string): string | null {
  if (typeof document === "undefined") {
    return null;
  }

  try {
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) {
      return parts.pop()?.split(';').shift() || null;
    }
    return null;
  } catch {
    return null;
  }
}

function isSessionTokenValid(): boolean {
  const token = readBrowserCookie(SESSION_JWT_COOKIE_NAME);
  if (!token) {
    return false;
  }

  // Check if token is expired
  return !isTokenExpired(token);
}

export interface AuthProviderProps {
  initialProfile: ProfileResponseDto | null;
  initialRoles: string[];
  initialPermissions: string[];
  shouldClearCache?: boolean;
  children: ReactNode;
}

export function AuthProvider({
  initialProfile,
  initialRoles,
  initialPermissions,
  shouldClearCache = false,
  children,
}: AuthProviderProps) {
  const [state, setState] = useState<AuthState>({
    profile: initialProfile,
    roles: sanitizeRoles(initialRoles),
    permissions: sanitizePermissions(initialPermissions),
  });
  const [isHydrated, setIsHydrated] = useState(false);

  // Sync with server-provided props or fall back to cached client state
  useEffect(() => {
    // If server signals to clear cache, do so immediately
    if (shouldClearCache) {
      console.info("[Auth] Server requested cache clear");
      window.sessionStorage.removeItem(STORAGE_KEY);
      setIsHydrated(true);
      return;
    }

    if (initialProfile) {
      // Only update state if it's different from initial values
      setState((prev) => {
        const needsUpdate =
          prev.profile?.email !== initialProfile.email ||
          prev.roles.length !== initialRoles.length ||
          prev.permissions.length !== initialPermissions.length;

        if (needsUpdate) {
          return {
            profile: initialProfile,
            roles: sanitizeRoles(initialRoles),
            permissions: sanitizePermissions(initialPermissions),
          };
        }
        return prev;
      });
      setIsHydrated(true);
      return;
    }

    // No server session - check if cached data is still valid
    const stored = readStoredState();
    if (stored?.profile) {
      // Validate session token before using cached data
      if (isSessionTokenValid()) {
        setState(stored);
      } else {
        // Token expired or missing - clear cache
        console.info("[Auth] Session token invalid, clearing cached auth state");
        window.sessionStorage.removeItem(STORAGE_KEY);
      }
    }

    setIsHydrated(true);
  }, [initialProfile, initialRoles, initialPermissions, shouldClearCache]);

  // Persist whenever state changes (and after hydration to avoid SSR mismatch)
  useEffect(() => {
    if (!isHydrated) return;
    persistState(state);
  }, [state, isHydrated]);

  const value = useMemo<AuthContextValue>(() => {
    const { profile, roles, permissions } = state;

    return {
      profile,
      roles,
      permissions,
      hasPermission: (permission: string) => checkPermission(permissions, permission),
      hasAnyPermission: (required: string[]) => checkAnyPermission(permissions, required),
      hasAllPermissions: (required: string[]) => checkAllPermissions(permissions, required),
      hasRole: (role: string) => checkRole(roles, role),
      hasAnyRole: (required: string[]) => checkAnyRole(roles, required),
      hasAllRoles: (required: string[]) => checkAllRoles(roles, required),
      isInitialized: isHydrated,
      isAuthenticated: !!profile,
      updateAuthState: (next: Partial<AuthState>) =>
        setState((prev) => ({
          profile:
            next.profile === undefined ? prev.profile : next.profile ?? null,
          roles:
            next.roles === undefined
              ? prev.roles
              : sanitizeRoles(next.roles),
          permissions:
            next.permissions === undefined
              ? prev.permissions
              : sanitizePermissions(next.permissions),
        })),
      clearAuthState: () => {
        // Clear sessionStorage
        if (typeof window !== "undefined") {
          window.sessionStorage.removeItem(STORAGE_KEY);
        }
        // Reset state
        setState({
          profile: null,
          roles: [],
          permissions: [],
        });
      },
    };
  }, [state, isHydrated]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuthContext(): AuthContextValue | null {
  return useContext(AuthContext);
}
