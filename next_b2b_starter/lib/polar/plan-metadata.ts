/**
 * Coerces string-typed numeric product metadata ("500") to numbers so
 * configured quotas render instead of "—" and checkouts forward numeric
 * values. Returns null for missing, empty, or non-numeric values.
 */
export function coerceNumericMetadata(value: unknown): number | null {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === "string" && value.trim() !== "") {
    const parsed = Number(value);
    if (Number.isFinite(parsed)) {
      return parsed;
    }
  }
  return null;
}
