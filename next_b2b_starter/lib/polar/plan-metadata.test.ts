import { describe, expect, it } from "vitest";
import { coerceNumericMetadata } from "./plan-metadata";

describe("coerceNumericMetadata", () => {
  it("coerces string-typed numeric metadata (\"500\"-style values)", () => {
    expect(coerceNumericMetadata("500")).toBe(500);
    expect(coerceNumericMetadata(" 250 ")).toBe(250);
  });

  it("passes numeric values through unchanged", () => {
    expect(coerceNumericMetadata(500)).toBe(500);
    expect(coerceNumericMetadata(0)).toBe(0);
  });

  it("returns null for missing, empty, or non-numeric values", () => {
    expect(coerceNumericMetadata(null)).toBeNull();
    expect(coerceNumericMetadata(undefined)).toBeNull();
    expect(coerceNumericMetadata("")).toBeNull();
    expect(coerceNumericMetadata("  ")).toBeNull();
    expect(coerceNumericMetadata("abc")).toBeNull();
    expect(coerceNumericMetadata("12x")).toBeNull();
  });
});
