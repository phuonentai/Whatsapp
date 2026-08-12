"use client";

import { useState } from "react";
import { useSearchParams } from "next/navigation";
import { ui } from "@/lib/copy/ui";
import { SuppliersManager } from "./suppliers-manager";
import { ProductsManager } from "./products-manager";
import { RunWizard } from "./run-wizard";
import { RunBoard } from "./run-board";

type Tab = "proveedores" | "productos" | "cotizaciones";

const TABS: { id: Tab; label: string }[] = [
  { id: "proveedores", label: ui.procurement.tabSuppliers },
  { id: "productos", label: ui.procurement.tabProducts },
  { id: "cotizaciones", label: ui.procurement.tabRuns },
];

export function ProcurementPage() {
  const searchParams = useSearchParams();
  const runId = Number(searchParams.get("run")) || 0;
  const [tab, setTab] = useState<Tab>("proveedores");

  // Deep link to a run board opens the board above the tabs.
  if (runId > 0) {
    return (
      <div className="space-y-4">
        <a href="/dashboard/procurement" className="text-sm text-blue-600 hover:underline">
          ← {ui.procurement.runsTitle}
        </a>
        <RunBoard runId={runId} />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">{ui.procurement.title}</h1>
      </div>
      <div className="flex gap-1 border-b">
        {TABS.map((t) => (
          <button
            key={t.id}
            onClick={() => setTab(t.id)}
            className={`-mb-px border-b-2 px-4 py-2 text-sm font-medium ${
              tab === t.id
                ? "border-blue-600 text-blue-600"
                : "border-transparent text-gray-500 hover:text-gray-800"
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>
      {tab === "proveedores" && <SuppliersManager />}
      {tab === "productos" && <ProductsManager />}
      {tab === "cotizaciones" && <RunWizard />}
    </div>
  );
}
