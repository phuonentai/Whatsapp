import { create } from "zustand";

type InboxState = {
  lastSeenAt: Record<number, number>;
  markSeen: (conversationId: number) => void;
};

export const useInboxStore = create<InboxState>((set) => ({
  lastSeenAt: {},
  markSeen: (conversationId) =>
    set((state) => ({
      lastSeenAt: { ...state.lastSeenAt, [conversationId]: Date.now() },
    })),
}));
