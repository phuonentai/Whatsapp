"use client";

import { AlertTriangle, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";

interface ErrorStateProps {
  title: string;
  description?: string;
  onRetry?: () => void;
  isRetrying?: boolean;
  retryLabel?: string;
}

export function ErrorState({
  title,
  description,
  onRetry,
  isRetrying = false,
  retryLabel = "Reintentar",
}: ErrorStateProps) {
  return (
    <div
      role="alert"
      className="flex flex-col items-center justify-center gap-3 rounded-lg border border-red-200 bg-red-50 px-6 py-8 text-center"
    >
      <AlertTriangle className="h-8 w-8 text-red-500" />
      <div className="space-y-1">
        <p className="text-sm font-medium text-red-800">{title}</p>
        {description && <p className="text-sm text-red-600">{description}</p>}
      </div>
      {onRetry && (
        <Button variant="outline" size="sm" onClick={onRetry} disabled={isRetrying}>
          {isRetrying && <RefreshCw className="h-3.5 w-3.5 animate-spin" />}
          {isRetrying ? "Reintentando..." : retryLabel}
        </Button>
      )}
    </div>
  );
}
