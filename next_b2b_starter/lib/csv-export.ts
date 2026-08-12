"use client";

import { useCallback, useState } from "react";
import { toast } from "sonner";
import { apiClient, resolveAccessToken } from "@/lib/api/api/client/api-client";

// CSV endpoints stream a file, so they cannot use the JSON unwrapping ApiClient.
// Fetch + blob carries the Stytch session token in the request headers (a bare
// window.location navigation cannot attach it).
export async function downloadCsvFile(endpoint: string, filename: string): Promise<void> {
  const token = await resolveAccessToken();
  const headers: Record<string, string> = token ? { Authorization: `Bearer ${token}` } : {};
  const response = await fetch(`${apiClient.getBaseUrl()}${endpoint}`, {
    headers,
    credentials: "include",
  });
  if (!response.ok) {
    throw new Error(`API Error ${response.status}: no se pudo descargar el archivo`);
  }
  const blob = await response.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}

interface CsvExportOptions {
  run: () => Promise<void>;
  successMessage: string;
  errorMessage: string;
}

/**
 * Shared export flow: every list view that exports CSV follows the same
 * pattern (loading flag, success toast, Spanish error toast). Keeps the
 * per-view handlers from being copy-pasted.
 */
export function useCsvExport({ run, successMessage, errorMessage }: CsvExportOptions) {
  const [isExporting, setIsExporting] = useState(false);
  const handleExport = useCallback(async () => {
    setIsExporting(true);
    try {
      await run();
      toast.success(successMessage);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : errorMessage);
    } finally {
      setIsExporting(false);
    }
  }, [run, successMessage, errorMessage]);
  return { isExporting, handleExport };
}
