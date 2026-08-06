"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { usePermissions } from "@/lib/hooks/use-permissions";

export default function NotFound() {
  const router = useRouter();
  const { profile, isInitialized } = usePermissions();

  useEffect(() => {
    if (!isInitialized) return;

    // Redirect authenticated users to dashboard, others to landing page
    const redirectTo = profile ? "/dashboard" : "/";
    router.replace(redirectTo);
  }, [profile, isInitialized, router]);

  // Show minimal loading state while redirecting
  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="h-8 w-8 animate-spin rounded-full border-4 border-slate-200 border-t-[#0FA8A0]" />
    </div>
  );
}
