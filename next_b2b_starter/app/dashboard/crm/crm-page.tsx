"use client";

import { useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useEntitlementQuery, useFeatures } from "@/lib/hooks/use-entitlement";
import { ContactTable } from "@/components/crm/contact-table";
import { CompanyTable } from "@/components/crm/company-table";
import { DealKanban } from "@/components/crm/deal-kanban";
import { ActivityTimeline } from "@/components/crm/activity-timeline";
import { PipelineEditor } from "@/components/crm/pipeline-editor";
import { TagManager } from "@/components/crm/tag-manager";
import { UpgradeBanner } from "@/components/crm/upgrade-banner";

type Tab = "contactos" | "empresas" | "negocios" | "actividad" | "etiquetas";

const TAB_LABELS: Record<Tab, { label: string; feature?: string; upgradePlan?: string }> = {
  contactos: { label: "Contactos" },
  empresas: { label: "Empresas", feature: "crm_companies", upgradePlan: "Pro" },
  negocios: { label: "Negocios", feature: "crm_deals", upgradePlan: "Pro" },
  actividad: { label: "Actividad", feature: "crm_activities", upgradePlan: "Pro" },
  etiquetas: { label: "Etiquetas", feature: "crm_tags", upgradePlan: "Enterprise" },
};

export function CRMPage() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const view = (searchParams.get("view") as Tab) || "contactos";
  const features = useFeatures();
  const { data: entitlement, isLoading } = useEntitlementQuery();

  const setView = (tab: Tab) => {
    const params = new URLSearchParams(searchParams);
    params.set("view", tab);
    router.replace(`/dashboard/crm?${params.toString()}`);
  };

  const tabs: { key: Tab; label: string; disabled?: boolean; upgradePlan?: string }[] = Object.entries(TAB_LABELS)
    .filter(([key]) => {
      if (key === "contactos") return true;
      const tab = key as Tab;
      return entitlement?.funcionalidades?.[TAB_LABELS[tab].feature!] !== false;
    })
    .map(([key, cfg]) => {
      const tab = key as Tab;
      const enabled = tab === "contactos" || entitlement?.funcionalidades?.[cfg.feature!] === true;
      return { key: tab, label: cfg.label, disabled: !enabled, upgradePlan: cfg.upgradePlan };
    });

  if (isLoading) return <div className="p-8 text-gray-500">Cargando CRM...</div>;

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-4">CRM</h1>
      {entitlement?.solo_lectura && (
        <div className="bg-yellow-100 border border-yellow-400 text-yellow-800 px-4 py-3 rounded mb-4">
          Tu suscripción está en modo de solo lectura. Reactívala para hacer cambios.
        </div>
      )}
      {entitlement?.periodo_gracia && (
        <div className="bg-orange-100 border border-orange-400 text-orange-800 px-4 py-3 rounded mb-4">
          Tu suscripción está en periodo de gracia. Actualiza tu método de pago para evitar la suspensión.
        </div>
      )}

      <div className="flex gap-2 mb-6 border-b pb-2">
        {tabs.map((tab) => (
          <button
            key={tab.key}
            onClick={() => !tab.disabled && setView(tab.key)}
            className={`px-4 py-2 rounded-t text-sm font-medium ${
              view === tab.key
                ? "bg-blue-600 text-white"
                : tab.disabled
                ? "text-gray-400 cursor-not-allowed"
                : "text-gray-700 hover:bg-gray-100"
            }`}
            disabled={tab.disabled}
          >
            {tab.label}
            {tab.disabled && tab.upgradePlan && (
              <span className="ml-1 text-xs text-gray-400">(Pro)</span>
            )}
          </button>
        ))}
      </div>

      {view === "contactos" && <ContactTable />}
      {view === "empresas" && (
        entitlement?.funcionalidades?.crm_companies ? <CompanyTable /> : <UpgradeBanner feature="Empresas" plan="Pro" />
      )}
      {view === "negocios" && (
        entitlement?.funcionalidades?.crm_deals ? <DealKanban /> : <UpgradeBanner feature="Negocios" plan="Pro" />
      )}
      {view === "actividad" && (
        entitlement?.funcionalidades?.crm_activities ? <ActivityTimeline /> : <UpgradeBanner feature="Actividad" plan="Pro" />
      )}
      {view === "etiquetas" && (
        entitlement?.funcionalidades?.crm_tags ? <TagManager /> : <UpgradeBanner feature="Etiquetas" plan="Enterprise" />
      )}
    </div>
  );
}
