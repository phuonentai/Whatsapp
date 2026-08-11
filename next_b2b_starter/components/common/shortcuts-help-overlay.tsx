"use client";

import { useEffect, useRef } from "react";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { globalShortcuts } from "@/lib/command-registry";
import { useShortcutsHelpStore } from "@/lib/stores/shortcuts-help-store";

export function ShortcutsHelpOverlay() {
  const { open, closeHelp } = useShortcutsHelpStore();
  const closeButtonRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (open) {
      window.setTimeout(() => closeButtonRef.current?.focus(), 0);
    }
  }, [open]);

  return (
    <Dialog open={open} onOpenChange={(next) => !next && closeHelp()}>
      <DialogContent
        className="sm:max-w-md"
        hideCloseButton
      >
        <DialogHeader>
          <DialogTitle className="text-xl font-semibold text-foreground">
            Keyboard shortcuts
          </DialogTitle>
          <DialogDescription className="text-sm text-muted-foreground">
            Move around the workspace faster without touching the mouse.
          </DialogDescription>
        </DialogHeader>
        <ul className="space-y-2">
          {globalShortcuts.map((shortcut) => (
            <li
              key={shortcut.id}
              className="flex items-center justify-between gap-4 rounded-md px-2 py-1.5 text-sm"
            >
              <span className="text-foreground">{shortcut.label}</span>
              <span className="inline-flex items-center gap-1">
                {shortcut.keys.map((key, index) => (
                  <span key={index}>
                    {index > 0 && (
                      <span className="mx-0.5 text-muted-foreground">then</span>
                    )}
                    <kbd className="inline-flex h-6 min-w-6 select-none items-center justify-center rounded border border-border bg-muted px-1.5 font-mono text-xs font-medium text-foreground">
                      {key}
                    </kbd>
                  </span>
                ))}
              </span>
            </li>
          ))}
        </ul>
        <button
          ref={closeButtonRef}
          type="button"
          onClick={closeHelp}
          className="mt-2 inline-flex h-9 items-center justify-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
        >
          Close
        </button>
      </DialogContent>
    </Dialog>
  );
}
