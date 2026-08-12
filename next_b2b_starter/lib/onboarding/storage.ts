import type { SignupBusinessContext } from "@/lib/models/signup.model";

export const ONBOARDING_STORAGE_KEYS = {
  businessContext: "ai-onboarding.business-context",
  assistantIntroDismissed: "ai-onboarding.assistant-intro-dismissed",
  inboxVisited: "ai-onboarding.inbox-visited",
  knowledgeVisited: "ai-onboarding.knowledge-visited",
} as const;

function readJson<T>(key: string): T | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = window.localStorage.getItem(key);
    return raw ? (JSON.parse(raw) as T) : null;
  } catch {
    return null;
  }
}

function write(key: string, value: string): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(key, value);
  } catch {
    // Private mode / quota: context is only a priority hint, never a gate.
  }
}

function readFlag(key: string): boolean {
  return readJson<boolean>(key) === true;
}

export function loadBusinessContext(): SignupBusinessContext | null {
  return readJson<SignupBusinessContext>(ONBOARDING_STORAGE_KEYS.businessContext);
}

export function saveBusinessContext(context: SignupBusinessContext): void {
  write(ONBOARDING_STORAGE_KEYS.businessContext, JSON.stringify(context));
}

export function isAssistantIntroDismissed(): boolean {
  return readFlag(ONBOARDING_STORAGE_KEYS.assistantIntroDismissed);
}

export function dismissAssistantIntro(): void {
  write(ONBOARDING_STORAGE_KEYS.assistantIntroDismissed, JSON.stringify(true));
}

export function isInboxVisited(): boolean {
  return readFlag(ONBOARDING_STORAGE_KEYS.inboxVisited);
}

export function markInboxVisited(): void {
  write(ONBOARDING_STORAGE_KEYS.inboxVisited, JSON.stringify(true));
}

export function isKnowledgeVisited(): boolean {
  return readFlag(ONBOARDING_STORAGE_KEYS.knowledgeVisited);
}

export function markKnowledgeVisited(): void {
  write(ONBOARDING_STORAGE_KEYS.knowledgeVisited, JSON.stringify(true));
}
