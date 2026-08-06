import "server-only";
import Stytch, { envs as stytchEnvs } from "stytch";

type B2BClient = InstanceType<typeof Stytch.B2BClient>;

let client: B2BClient | null = null;

function requiredEnv(name: string): string {
  const value = process.env[name];
  if (!value) {
    throw new Error(`Missing ${name} environment variable for Stytch configuration.`);
  }
  return value;
}

export function getStytchB2BClient(): B2BClient {
  if (client) return client;

  const projectId = requiredEnv("STYTCH_PROJECT_ID");
  const secret = requiredEnv("STYTCH_SECRET");
  const projectEnv = process.env.STYTCH_PROJECT_ENV || process.env.NEXT_PUBLIC_STYTCH_PROJECT_ENV || "test";

  client = new Stytch.B2BClient({
    project_id: projectId,
    secret,
    env: projectEnv === "live" ? stytchEnvs.live : stytchEnvs.test,
  });

  return client;
}
