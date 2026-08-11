import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { ui, en } from "./ui";

function collectStrings(node: unknown, out: string[] = []): string[] {
  if (typeof node === "string") {
    out.push(node);
    return out;
  }
  if (Array.isArray(node)) {
    for (const item of node) collectStrings(item, out);
    return out;
  }
  if (node && typeof node === "object") {
    for (const value of Object.values(node)) collectStrings(value, out);
  }
  return out;
}

describe("copy layer", () => {
  it("every Spanish key is non-empty (never resolves to fallback)", () => {
    const spanish = collectStrings(ui);
    expect(spanish.length).toBeGreaterThan(0);
    for (const value of spanish) {
      expect(value.trim().length, `empty Spanish string: ${JSON.stringify(value)}`).toBeGreaterThan(0);
    }
  });

  it("primary flows (auth, billing, whatsapp) ship fully in Spanish", () => {
    for (const ns of ["auth", "billing", "whatsapp"] as const) {
      const spanish = collectStrings(ui[ns]);
      for (const value of spanish) {
        expect(
          value.trim().length,
          `[${ns}] resolves to empty value: ${JSON.stringify(value)}`
        ).toBeGreaterThan(0);
      }
    }
  });

  it("English fallback values are never empty", () => {
    const english = collectStrings(en);
    for (const value of english) {
      expect(value.trim().length, `empty English fallback: ${JSON.stringify(value)}`).toBeGreaterThan(0);
    }
  });
});

const HARDCODED_STRINGS = [
  "Choose your plan",
  "Scale approvals without hitting limits",
  "Loading plans",
  "Failed to load plans",
  "No plans available",
  "Processing your checkout",
  "Current Plan",
  "International card",
  "PSE / Nequi / Colombian card",
  "seats included",
  "invoices per month",
  "An unexpected error occurred",
  "Check your email",
  "Back to Sign In",
  "Create your account",
  "Full Name",
  "Organization Name",
  "Already have an account?",
  "No messages yet",
  "No conversation selected",
  "Open Inbox",
  "Manage CRM",
  "Knowledge Base",
  "Workspace settings",
  "Open conversations",
  "Contacts",
  "Deals",
  "Deals by stage",
  "Recent activity",
  "Quick actions",
  "No Instagram messages yet",
  "No WhatsApp messages yet",
  "No conversations found",
  "Connecting your WhatsApp",
  "Verifying Coexistence session",
  "Establishing secure token",
  "All set! Your WhatsApp is live",
  "WhatsApp connected",
  "Business phone",
  "Phone number ID",
  "WhatsApp Business Integration",
  "Subscription access restricted",
  "View pricing plans",
  "Talk to sales",
  "Invoices remaining",
  "Invoices used this period",
  "No active plan",
  "Browse plans",
  "Current plan",
  "Renews on",
  "Scheduled to end",
  "Resume subscription",
  "Schedule cancellation",
  "Refresh status",
  "Confirm cancellation",
  "Processing request",
  "Check your email",
];

const SWEPT_FILES = [
  "app/signup/page.tsx",
  "components/billing/plans-modal.tsx",
  "components/billing/subscription-paywall.tsx",
  "app/dashboard/settings/components/subscription-tab.tsx",
  "app/dashboard/settings/components/whatsapp-config-section.tsx",
  "app/dashboard/components/dashboard-home.tsx",
  "app/dashboard/inbox/components/conversation-list.tsx",
  "app/dashboard/inbox/components/message-thread.tsx",
  "app/dashboard/inbox/components/empty-state.tsx",
  "app/dashboard/inbox/components/agent-suggestions-panel.tsx",
  "app/dashboard/settings/components/agent-settings-section.tsx",
];

describe("hardcoded user-facing strings sweep", () => {
  it.each(SWEPT_FILES)("%s has no hardcoded English copy", (relative) => {
    const source = readFileSync(resolve(__dirname, "../..", relative), "utf-8");
    const hits = HARDCODED_STRINGS.filter((phrase) => {
      if (phrase === "Contacts" || phrase === "Deals") {
        return new RegExp(`[>"']\\s*${phrase}\\s*[<]`).test(source);
      }
      return source.includes(phrase);
    });
    expect(hits, `${relative} still contains hardcoded copy: ${hits.join(", ")}`).toEqual([]);
  });
});
