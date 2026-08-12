import { create } from "zustand";

export type CommandPaletteMode = "command" | "search";

interface CommandPaletteState {
  open: boolean;
  mode: CommandPaletteMode;
  session: number;
  /**
   * Monotonic counter consumed by the knowledge page to reset to a fresh
   * chat. The palette's "Nueva conversación de IA" action increments it; the
   * knowledge page replays the request on its next mount (a boolean would be
   * lost if the page was unmounted when the action ran).
   */
  aiNewChatSignal: number;
  openPalette: (mode?: CommandPaletteMode) => void;
  closePalette: () => void;
  requestNewAiChat: () => void;
}

export const useCommandPaletteStore = create<CommandPaletteState>((set) => ({
  open: false,
  mode: "command",
  session: 0,
  aiNewChatSignal: 0,
  openPalette: (mode = "command") =>
    set((state) => ({ open: true, mode, session: state.session + 1 })),
  closePalette: () => set({ open: false }),
  requestNewAiChat: () =>
    set((state) => ({ aiNewChatSignal: state.aiNewChatSignal + 1 })),
}));
