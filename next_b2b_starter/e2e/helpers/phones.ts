let phoneSeq = 0;

/**
 * Generate a valid Colombian mobile E.164 phone number that satisfies the
 * contact schema regex `^\+573\d{9}$` (exactly 9 digits after +573).
 * Unique per call within a serial test run (timestamp base + 2-digit seq).
 */
export function uniqueColombianPhone(): string {
  phoneSeq = (phoneSeq + 1) % 100;
  return `+573${String(Date.now()).slice(-7)}${String(phoneSeq).padStart(2, "0")}`;
}
