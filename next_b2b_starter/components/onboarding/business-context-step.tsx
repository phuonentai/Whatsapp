"use client";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ArrowRight, Check } from "lucide-react";
import { ui } from "@/lib/copy/ui";
import { saveBusinessContext } from "@/lib/onboarding/storage";
import type {
  SignupBusinessContext,
  WhatsAppReadiness,
} from "@/lib/models/signup.model";

const READINESS_OPTIONS: Array<{
  value: WhatsAppReadiness;
  label: string;
}> = [
  { value: "already", label: ui.onboarding.readinessAlready },
  { value: "planning", label: ui.onboarding.readinessPlanning },
  { value: "no", label: ui.onboarding.readinessNo },
];

interface BusinessContextStepProps {
  value: SignupBusinessContext;
  onChange: (updates: Partial<SignupBusinessContext>) => void;
  onBack: () => void;
  onContinue: () => void;
  canContinue: boolean;
  disabled?: boolean;
  submitLabel?: string;
}

export function BusinessContextStep({
  value,
  onChange,
  onBack,
  onContinue,
  canContinue,
  disabled,
  submitLabel = ui.auth.continue,
}: BusinessContextStepProps) {
  const handleContinue = () => {
    saveBusinessContext(value);
    onContinue();
  };

  return (
    <div className="space-y-5">
      <div>
        <h2 className="text-lg font-semibold text-foreground">
          {ui.onboarding.businessTitle}
        </h2>
        <p className="text-sm text-muted-foreground mt-1">
          {ui.onboarding.businessSubtitle}
        </p>
      </div>

      <fieldset>
        <legend className="block text-sm font-medium text-muted-foreground mb-2">
          {ui.onboarding.whatsappReadinessLabel}
        </legend>
        <p className="mb-3 text-xs text-muted-foreground">
          {ui.onboarding.whatsappReadinessHint}
        </p>
        <div className="space-y-2">
          {READINESS_OPTIONS.map((option) => {
            const selected = value.whatsappReadiness === option.value;
            return (
              <button
                key={option.value}
                type="button"
                role="radio"
                aria-checked={selected}
                onClick={() => onChange({ whatsappReadiness: option.value })}
                disabled={disabled}
                className={`flex w-full items-center justify-between gap-3 rounded-xl border px-4 py-3 text-left text-sm transition-colors ${
                  selected
                    ? "border-primary bg-primary/10 text-foreground"
                    : "border-border text-muted-foreground hover:border-primary/40 hover:bg-muted"
                }`}
              >
                <span>{option.label}</span>
                {selected ? (
                  <Check className="h-4 w-4 flex-none text-primary" aria-hidden />
                ) : null}
              </button>
            );
          })}
        </div>
      </fieldset>

      <div>
        <label className="block text-sm font-medium text-muted-foreground mb-1">
          {ui.onboarding.businessGoalLabel}
        </label>
        <Input
          type="text"
          placeholder={ui.onboarding.businessGoalPlaceholder}
          value={value.businessGoal}
          onChange={(e) => onChange({ businessGoal: e.target.value })}
          disabled={disabled}
          className="border-border bg-background text-foreground placeholder:text-muted-foreground focus:border-primary focus:ring-primary/30"
        />
        <p className="mt-1 text-xs text-muted-foreground">
          {ui.onboarding.businessGoalHint}
        </p>
      </div>

      <div className="flex gap-3 pt-2">
        <Button
          variant="outline"
          onClick={onBack}
          disabled={disabled}
          className="flex-1 border-border text-muted-foreground hover:bg-muted hover:text-foreground"
        >
          {ui.common.back}
        </Button>
        <Button
          onClick={handleContinue}
          disabled={!canContinue || disabled}
          className="flex-1 bg-primary hover:bg-primary/90"
        >
          {submitLabel} <ArrowRight className="ml-2 h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}
