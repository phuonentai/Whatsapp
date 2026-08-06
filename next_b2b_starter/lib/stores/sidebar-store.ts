import { create } from "zustand";

type SidebarState = {
  isCollapsed: boolean;
  preferredCollapsed: boolean;
  isAutoCollapsed: boolean;
  toggle: () => void;
  setCollapsed: (collapsed: boolean) => void;
  setAutoCollapsed: (shouldCollapse: boolean) => void;
};

export const useSidebarStore = create<SidebarState>((set, get) => ({
  isCollapsed: false,
  preferredCollapsed: false,
  isAutoCollapsed: false,
  toggle: () => {
    const { isAutoCollapsed } = get();

    if (isAutoCollapsed) {
      return;
    }

    set((state) => {
      const next = !state.preferredCollapsed;
      return {
        preferredCollapsed: next,
        isCollapsed: next,
      };
    });
  },
  setCollapsed: (collapsed) =>
    set((state) => ({
      preferredCollapsed: collapsed,
      isCollapsed: state.isAutoCollapsed ? true : collapsed,
    })),
  setAutoCollapsed: (shouldCollapse) =>
    set((state) => ({
      isAutoCollapsed: shouldCollapse,
      isCollapsed: shouldCollapse ? true : state.preferredCollapsed,
    })),
}));
