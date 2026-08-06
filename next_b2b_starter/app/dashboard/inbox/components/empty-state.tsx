"use client";

import Link from "next/link";
import { MessageCircle } from "lucide-react";

export function EmptyState() {
  return (
    <div className="flex h-full flex-col items-center justify-center px-6 text-center">
      <div className="flex h-16 w-16 items-center justify-center rounded-full bg-gray-100">
        <MessageCircle className="h-8 w-8 text-gray-400" />
      </div>
      <h3 className="mt-4 text-lg font-semibold text-gray-900">No conversation selected</h3>
      <p className="mt-2 max-w-sm text-sm text-gray-500">
        Select a conversation from the list to view messages, or connect WhatsApp in
        Settings to start receiving messages.
      </p>
      <Link
        href="/dashboard/settings?view=whatsapp"
        className="mt-6 text-sm font-medium text-blue-600 hover:text-blue-700"
      >
        Go to WhatsApp settings →
      </Link>
    </div>
  );
}
