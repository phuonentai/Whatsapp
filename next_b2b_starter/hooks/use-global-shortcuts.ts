"use client";

import { useCallback, useEffect, useRef } from "react";
import { useRouter } from "next/navigation";

import { globalShortcuts } from "@/lib/command-registry";
import { useCommandPaletteStore } from "@/lib/stores/command-palette-store";
import { useShortcutsHelpStore } from "@/lib/stores/shortcuts-help-store";

/**
 * Returns true when the event target is an editable surface where single-key
 * shortcuts must be suppressed (spec: input/textarea/contenteditable).
 */
export function isTypingTarget(target: EventTarget | null): boolean {
  if (!target || typeof target !== "object" || !("tagName" in target)) {
    return false;
  }
  const node = target as HTMLElement;
  const tag = node.tagName;
  if (tag === "INPUT" || tag === "TEXTAREA") {
    return true;
  }
  if (node.getAttribute("contenteditable") != null) {
    return true;
  }
  return Boolean(node.isContentEditable);
}

/**
 * Global keyboard shortcuts for the dashboard shell:
 * - `g d|i|c|k|s` navigates to a destination (see globalShortcuts registry)
 * - `?` opens the shortcuts help overlay
 * - Ctrl/Cmd+K toggles the command palette
 *
 * Single-key handling is suppressed while typing in an input, textarea, or
 * contenteditable. Meta-key combos (⌘K) are always honored.
 */
export function useGlobalShortcuts() {
  const router = useRouter();
  const sequence = useRef<string[]>([]);

  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      const target = event.target as EventTarget | null;

      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        const palette = useCommandPaletteStore.getState();
        if (palette.open) {
          palette.closePalette();
        } else {
          palette.openPalette("command");
        }
        return;
      }

      if (isTypingTarget(target)) {
        return;
      }

      if (event.key === "?") {
        event.preventDefault();
        useShortcutsHelpStore.getState().openHelp();
        return;
      }

      const key = event.key.toLowerCase();
      const isPlainModifier =
        event.metaKey || event.ctrlKey || event.altKey || event.shiftKey;

      if (isPlainModifier) {
        sequence.current = [];
        return;
      }

      if (sequence.current[0] === "g") {
        const destination = globalShortcuts.find(
          (shortcut) => shortcut.url && shortcut.keys[1] === key
        );
        if (destination?.url) {
          event.preventDefault();
          sequence.current = [];
          router.push(destination.url);
          return;
        }
        sequence.current = [];
        return;
      }

      if (key === "g") {
        sequence.current = ["g"];
        return;
      }

      sequence.current = [];
    },
    [router]
  );

  useEffect(() => {
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);
}
