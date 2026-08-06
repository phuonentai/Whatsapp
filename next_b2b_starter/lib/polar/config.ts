const meterId = process.env.NEXT_PUBLIC_POLAR_METER_ID ?? null;

export const POLAR_METER_ID = meterId;

/**
 * Check if Polar billing is enabled
 *
 * Polar is enabled if the POLAR_ACCESS_TOKEN environment variable is set.
 * Products are fetched dynamically from Polar API.
 */
export function isPolarEnabled(): boolean {
  return Boolean(process.env.POLAR_ACCESS_TOKEN);
}
