"use client";

import { ui } from "@/lib/copy/ui";
import { SchedulesManager } from "./schedules-manager";
import { FollowUpSettingsPanel } from "./followup-settings-panel";

/**
 * Schedules page (add-scheduled-inquiry-runs): org-scoped recurring inquiry
 * schedules with next-run visibility + the follow-up settings panel. Gated by
 * the sidebar entry (org:manage), same as the procurement section.
 */
export function SchedulesPage() {
  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-2xl font-semibold">{ui.procurement.schedulesTitle}</h1>
      </div>
      <SchedulesManager />
      <FollowUpSettingsPanel />
    </div>
  );
}
