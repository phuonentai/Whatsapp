/**
 * UI Store (Zustand)
 *
 * Manages client-only UI state such as:
 * - Active tabs
 * - Sidebar visibility
 * - Modal states
 * - UI preferences
 *
 * This store does NOT handle server data (use TanStack Query for that).
 */

import { create } from "zustand";

interface UIState {
  // Settings page active tab
  activeSettingsTab: string;
  setActiveSettingsTab: (tab: string) => void;

  // Sidebar state (for future use)
  isSidebarOpen: boolean;
  toggleSidebar: () => void;
  setSidebarOpen: (isOpen: boolean) => void;

  // Modal states (for future use)
  isPlansModalOpen: boolean;
  setPlansModalOpen: (isOpen: boolean) => void;

  // Reset all UI state
  reset: () => void;
}

const initialState = {
  activeSettingsTab: "profile",
  isSidebarOpen: true,
  isPlansModalOpen: false,
};

export const useUIStore = create<UIState>((set) => ({
  ...initialState,

  // Settings tab
  setActiveSettingsTab: (tab) => set({ activeSettingsTab: tab }),

  // Sidebar
  toggleSidebar: () =>
    set((state) => ({ isSidebarOpen: !state.isSidebarOpen })),
  setSidebarOpen: (isOpen) => set({ isSidebarOpen: isOpen }),

  // Modals
  setPlansModalOpen: (isOpen) => set({ isPlansModalOpen: isOpen }),

  // Reset
  reset: () => set(initialState),
}));

/**
 * Selector hooks for optimized re-renders
 *
 * Use these instead of the base store to prevent unnecessary re-renders.
 *
 * @example
 * ```tsx
 * // Bad: Component re-renders on ANY state change
 * const { activeSettingsTab } = useUIStore();
 *
 * // Good: Component re-renders only when activeSettingsTab changes
 * const activeSettingsTab = useActiveSettingsTab();
 * ```
 */

export const useActiveSettingsTab = () =>
  useUIStore((state) => state.activeSettingsTab);

export const useIsSidebarOpen = () =>
  useUIStore((state) => state.isSidebarOpen);

export const useIsPlansModalOpen = () =>
  useUIStore((state) => state.isPlansModalOpen);
