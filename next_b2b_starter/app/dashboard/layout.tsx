import type { ReactNode } from "react";
import { redirect } from "next/navigation";

import { DashboardLayout } from "@/components/layout/dashboard-layout";
import { resolveCurrentSubscription } from "@/lib/polar/current-subscription";

export default async function Layout({
  children,
}: {
  children: ReactNode;
}) {
  const subscription = await resolveCurrentSubscription();

  console.info("[Polar] Rendering dashboard layout", {
    isActive: subscription.isActive,
    reason: subscription.reason,
    status: subscription.status,
    backendAvailable: subscription.backendAvailable,
  });

  if (!subscription.isAuthenticated) {
    redirect("/auth");
  }

  return (
    <DashboardLayout initialSubscription={subscription}>
      {children}
    </DashboardLayout>
  );
}
