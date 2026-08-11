import { create } from "zustand";

export type CommandPaletteMode = "command" | "search";

interface CommandPaletteState {
  open: boolean;
  mode: CommandPaletteMode;
  session: number;
  openPalette: (mode?: CommandPaletteMode) => void;
  closePalette: () => void;
}

export const useCommandPaletteStore = create<CommandPaletteState>((set) => ({
  open: false,
  mode: "command",
  session: 0,
  openPalette: (mode = "command") =>
    set((state) => ({ open: true, mode, session: state.session + 1 })),
  closePalette: () => set({ open: false }),
}));
