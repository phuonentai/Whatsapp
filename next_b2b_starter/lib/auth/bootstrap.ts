import { getMemberSession } from "@/lib/auth/stytch/server";
import { getServerPermissions } from "@/lib/auth/server-permissions";

export async function authBootstrap() {
  const session = await getMemberSession();
  const permissions = await getServerPermissions(session);

  // Signal to clear cache if no valid session found
  const shouldClearCache = !permissions.profile;

  return {
    profile: permissions.profile,
    roles: permissions.roles,
    permissions: permissions.permissions,
    shouldClearCache,
  };
}
