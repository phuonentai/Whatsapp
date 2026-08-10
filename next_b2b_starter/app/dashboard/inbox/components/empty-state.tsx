"use client";

import Link from "next/link";
import { MessageCircle, Instagram } from "lucide-react";

interface EmptyStateProps {
  channel?: "all" | "whatsapp" | "instagram";
}

export function EmptyState({ channel = "all" }: EmptyStateProps) {
  const settingsView = channel === "instagram" ? "instagram" : "whatsapp";
  return (
    <div className="flex h-full flex-col items-center justify-center px-6 text-center">
      <div className="flex h-16 w-16 items-center justify-center rounded-full bg-gray-100">
        {channel === "instagram" ? (
          <Instagram className="h-8 w-8 text-gray-400" />
        ) : (
          <MessageCircle className="h-8 w-8 text-gray-400" />
        )}
      </div>
      <h3 className="mt-4 text-lg font-semibold text-gray-900">No conversation selected</h3>
      <p className="mt-2 max-w-sm text-sm text-gray-500">
        Select a conversation from the list to view messages, or connect a channel in
        Settings to start receiving messages.
      </p>
      <Link
        href={`/dashboard/settings?view=${settingsView}`}
        className="mt-6 text-sm font-medium text-blue-600 hover:text-blue-700"
      >
        Go to {channel === "instagram" ? "Instagram" : "WhatsApp"} settings →
      </Link>
    </div>
  );
}
