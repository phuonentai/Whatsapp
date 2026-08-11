import { create } from "zustand";

interface ShortcutsHelpState {
  open: boolean;
  openHelp: () => void;
  closeHelp: () => void;
}

export const useShortcutsHelpStore = create<ShortcutsHelpState>((set) => ({
  open: false,
  openHelp: () => set({ open: true }),
  closeHelp: () => set({ open: false }),
}));
