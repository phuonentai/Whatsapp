"use client";

import { useState } from "react";
import { Check } from "lucide-react";
import { cn } from "@/lib/utils";
import { copy } from "@/lib/copy/ui";

/**
 * Checklist interactivo "Antes de empezar" de la página /onboarding-info
 * (plantilla Verifika OnboardingInfo S4). Sin persistencia: estado local.
 */
export function OnboardingInfoChecklist() {
  const items = copy("marketing", "onboardingInfoChecklistItems");
  const [checked, setChecked] = useState<boolean[]>(() => items.map(() => false));

  const done = checked.filter(Boolean).length;
  const progress = items.length > 0 ? Math.round((done / items.length) * 100) : 0;

  const toggle = (index: number) =>
    setChecked((prev) => prev.map((value, i) => (i === index ? !value : value)));

  return (
    <div className="rounded-2xl border border-slate-200 bg-white p-6 lg:p-8">
      <div className="flex items-center justify-between gap-4 mb-4">
        <p className="text-sm font-semibold text-slate-900">
          {copy("marketing", "onboardingInfoChecklistBody")}
        </p>
        <span className="text-sm font-bold tabular-nums text-emerald-600 shrink-0">
          {progress}%
        </span>
      </div>
      <div className="h-2 rounded-full bg-slate-100 mb-6" role="progressbar" aria-valuenow={progress} aria-valuemin={0} aria-valuemax={100}>
        <div
          className="h-2 rounded-full bg-emerald-500 transition-all duration-300"
          style={{ width: `${progress}%` }}
        />
      </div>
      <ul className="space-y-3">
        {items.map((item, index) => (
          <li key={item}>
            <label className="flex items-start gap-3 cursor-pointer group">
              <input
                type="checkbox"
                checked={checked[index]}
                onChange={() => toggle(index)}
                className="sr-only"
              />
              <span
                className={cn(
                  "mt-0.5 w-5 h-5 rounded-md border flex items-center justify-center shrink-0 transition-colors",
                  checked[index]
                    ? "bg-emerald-500 border-emerald-500 text-white"
                    : "border-slate-300 bg-white group-hover:border-emerald-500"
                )}
                aria-hidden="true"
              >
                {checked[index] && <Check className="w-3.5 h-3.5" />}
              </span>
              <span
                className={cn(
                  "text-sm leading-relaxed",
                  checked[index] ? "text-slate-400 line-through" : "text-slate-700"
                )}
              >
                {item}
              </span>
            </label>
          </li>
        ))}
      </ul>
    </div>
  );
}
