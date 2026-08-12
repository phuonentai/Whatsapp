"use client";

import { useState } from "react";
import Link from "next/link";
import {
  ArrowRight,
  CheckCircle2,
  Inbox,
  MessageSquareText,
  ShieldCheck,
  Sparkles,
  X,
} from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { ui } from "@/lib/copy/ui";

export const POST_CONNECT_DISMISS_KEY = "whatsapp-post-connect-dismissed";

function readDismissedFlag(): boolean {
  if (typeof window === "undefined") return false;
  try {
    return window.localStorage.getItem(POST_CONNECT_DISMISS_KEY) === "true";
  } catch {
    // Private mode / quota: dismissal is only a UI hint, never a gate.
    return false;
  }
}

export function isPostConnectDismissed(): boolean {
  return readDismissedFlag();
}

export function dismissPostConnect(): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(POST_CONNECT_DISMISS_KEY, "true");
  } catch {
    // Private mode / quota: dismissal is only a UI hint, never a gate.
  }
}

export function clearPostConnectDismissed(): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.removeItem(POST_CONNECT_DISMISS_KEY);
  } catch {
    // Private mode / quota: dismissal is only a UI hint, never a gate.
  }
}

/**
 * Post-connect next-steps card. Rendered by the WhatsApp config surface after
 * a successful connect; dismissible per client via localStorage. Renders no
 * API calls of its own — the test-message path reuses the existing inbound
 * webhook flow.
 */
export function PostConnectSteps() {
  const [dismissed, setDismissed] = useState(isPostConnectDismissed);

  if (dismissed) {
    return null;
  }

  const handleDismiss = () => {
    dismissPostConnect();
    setDismissed(true);
  };

  return (
    <Card className="border-emerald-200 bg-emerald-50/60">
      <CardHeader className="flex-row items-start justify-between space-y-0 pb-2">
        <div className="flex items-start gap-3">
          <span className="flex h-10 w-10 flex-none items-center justify-center rounded-full bg-emerald-100">
            <CheckCircle2 className="h-5 w-5 text-primary" aria-hidden />
          </span>
          <div className="space-y-1">
            <CardTitle className="text-lg text-gray-900">
              {ui.whatsapp.postConnectTitle}
            </CardTitle>
          </div>
        </div>
        <Button
          variant="ghost"
          size="sm"
          className="flex-none px-2"
          onClick={handleDismiss}
          aria-label={ui.whatsapp.postConnectDismiss}
        >
          <X className="h-4 w-4 text-gray-500" aria-hidden />
        </Button>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex items-start gap-3">
          <MessageSquareText className="mt-0.5 h-4 w-4 flex-none text-primary" aria-hidden />
          <div>
            <p className="text-sm font-medium text-gray-900">
              {ui.whatsapp.postConnectTestMessage}
            </p>
            <p className="text-sm text-gray-500">{ui.whatsapp.postConnectTestMessageBody}</p>
          </div>
        </div>

        <div className="flex items-start gap-3">
          <ArrowRight className="mt-0.5 h-4 w-4 flex-none text-primary" aria-hidden />
          <div>
            <p className="text-sm font-medium text-gray-900">
              {ui.whatsapp.postConnectGoForward}
            </p>
            <p className="text-sm text-gray-500">{ui.whatsapp.postConnectGoForwardBody}</p>
          </div>
        </div>

        <div className="flex items-start gap-3">
          <ShieldCheck className="mt-0.5 h-4 w-4 flex-none text-primary" aria-hidden />
          <div>
            <p className="text-sm font-medium text-gray-900">{ui.whatsapp.postConnectConsent}</p>
            <p className="text-sm text-gray-500">{ui.whatsapp.postConnectConsentBody}</p>
            <Link
              href="/dashboard/settings?view=compliance"
              className="mt-1 inline-flex items-center gap-1 text-sm font-medium text-primary hover:text-emerald-800"
            >
              {ui.whatsapp.postConnectConsentLink}
            </Link>
          </div>
        </div>

        <div className="flex flex-wrap gap-2 pt-1">
          <Link href="/dashboard/inbox">
            <Button size="sm" variant="outline" className="bg-white">
              <Inbox className="mr-2 h-4 w-4" aria-hidden />
              {ui.whatsapp.postConnectInboxCta}
            </Button>
          </Link>
          <Link href="/dashboard/settings?view=ai">
            <Button size="sm">
              <Sparkles className="mr-2 h-4 w-4" aria-hidden />
              {ui.whatsapp.postConnectAssistantCta}
              <ArrowRight className="ml-2 h-4 w-4" aria-hidden />
            </Button>
          </Link>
        </div>
      </CardContent>
    </Card>
  );
}
