"use client";

import { useState } from "react";
import { useSignupFlow } from "@/hooks/use-signup-flow";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ArrowRight, Home, Inbox } from "lucide-react";
import { ui, tpl } from "@/lib/copy/ui";
import { BusinessContextStep } from "@/components/onboarding/business-context-step";
import Link from "next/link";
import { cn } from "@/lib/utils";

const STEP_INDEX: Record<string, number> = {
  account: 1,
  organization: 2,
  business: 3,
};

export default function SignupPage() {
  const {
    step,
    owner,
    organization,
    business,
    isLoading,
    error,
    emailSent,
    canContinueAccount,
    canContinueOrganization,
    canContinueBusiness,
    goNext,
    goBack,
    sendMagicLink,
    updateOwner,
    updateOrganization,
    updateBusiness,
  } = useSignupFlow();

  const stepIndex = STEP_INDEX[step] ?? 1;

  // Campos fiscales DIAN visuales (plantilla Verifika): sin persistencia ni
  // campos nuevos en el payload de signup (spec signup-stytch-compliance).
  const [visualNit, setVisualNit] = useState("");
  const [visualRegimen, setVisualRegimen] = useState<string>(ui.onboarding.wizardRegimenSimplificado);
  const [visualCity, setVisualCity] = useState<string>(ui.onboarding.wizardCities[0]);

  // Success view after email sent
  if (emailSent) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center px-6">
        <div className="w-full max-w-md text-center space-y-6 rounded-2xl border border-border bg-card p-8 shadow-lg shadow-slate-900/5">
          <div className="mx-auto h-14 w-14 bg-primary/10 rounded-full flex items-center justify-center">
            <Inbox className="h-7 w-7 text-primary" />
          </div>
          <h1 className="text-2xl font-semibold text-foreground">{ui.auth.emailSentTitle}</h1>
          <p className="text-muted-foreground">
            {ui.auth.emailSentIntro} <strong className="text-foreground">{owner.email}</strong>.{" "}
            {ui.auth.emailSentCta}
          </p>
          <Link href="/auth">
            <Button variant="outline" className="border-border text-muted-foreground hover:bg-muted hover:text-foreground">
              {ui.auth.backToSignIn}
            </Button>
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background flex items-center justify-center px-6 py-10">
      <div className="w-full max-w-md">
        {/* Wizard progress (onboarding.html composition) */}
        <div className="mb-6">
          <div className="flex items-center justify-between text-sm">
            <p className="text-muted-foreground">{tpl(ui.auth.stepProgress, { current: stepIndex, total: 3 })}</p>
            <Link
              href="/"
              className="flex items-center gap-1.5 text-xs text-muted-foreground transition-colors hover:text-primary"
            >
              <Home className="h-3.5 w-3.5" />
              <span>{ui.auth.home}</span>
            </Link>
          </div>
          <div className="mt-2.5 h-1.5 overflow-hidden rounded-full bg-muted">
            <div
              className="h-full rounded-full bg-primary transition-all duration-300"
              style={{ width: `${(stepIndex / 3) * 100}%` }}
            />
          </div>
        </div>

        <div className="rounded-2xl border border-border bg-card p-8 shadow-lg shadow-slate-900/5">
          <div className="flex items-center justify-between mb-5">
            <div>
              {/* Bienvenida Verifika (spec dashboard-template-restyle) */}
              <h1 className="text-2xl font-semibold text-foreground">
                {ui.onboarding.wizardWelcomeTitle}
              </h1>
              <p className="text-sm text-muted-foreground mt-1">
                {ui.onboarding.wizardWelcomeSubtitle}
              </p>
            </div>
          </div>

          {error && (
            <div className="mb-4 p-3 rounded-md border border-red-500/30 bg-red-500/10">
              <p className="text-sm text-red-600">{error}</p>
            </div>
          )}

          {/* Step 1: Account */}
          {step === "account" && (
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-muted-foreground mb-1">
                  {ui.auth.fullName}
                </label>
                <Input
                  type="text"
                  placeholder={ui.auth.fullNamePlaceholder}
                  value={owner.fullName}
                  onChange={(e) => updateOwner({ fullName: e.target.value })}
                  disabled={isLoading}
                  className="border-border bg-background text-foreground placeholder:text-muted-foreground focus:border-primary focus:ring-primary/30"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-muted-foreground mb-1">
                  {ui.auth.email}
                </label>
                <Input
                  type="email"
                  placeholder={ui.auth.emailPlaceholder}
                  value={owner.email}
                  onChange={(e) => updateOwner({ email: e.target.value })}
                  disabled={isLoading}
                  className="border-border bg-background text-foreground placeholder:text-muted-foreground focus:border-primary focus:ring-primary/30"
                />
              </div>
              <Button
                onClick={goNext}
                disabled={!canContinueAccount || isLoading}
                className="w-full bg-primary hover:bg-primary/90"
              >
                {ui.auth.continue} <ArrowRight className="ml-2 h-4 w-4" />
              </Button>
            </div>
          )}

          {/* Step 2: Organization */}
          {step === "organization" && (
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-muted-foreground mb-1">
                  {ui.auth.organizationName}
                </label>
                <Input
                  type="text"
                  placeholder={ui.auth.organizationNamePlaceholder}
                  value={organization.displayName}
                  onChange={(e) => updateOrganization({ displayName: e.target.value })}
                  disabled={isLoading}
                  className="border-border bg-background text-foreground placeholder:text-muted-foreground focus:border-primary focus:ring-primary/30"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-muted-foreground mb-1">
                  {ui.auth.industry}
                </label>
                <select
                  className={cn(
                    "w-full rounded-md border border-border bg-background px-3 py-2 text-foreground",
                    "focus:border-primary focus:ring-primary/30 focus:outline-none focus:ring-2"
                  )}
                  value={organization.industry}
                  onChange={(e) => updateOrganization({ industry: e.target.value })}
                  disabled={isLoading}
                >
                  <option value="Technology">{ui.auth.industryTechnology}</option>
                  <option value="Finance">{ui.auth.industryFinance}</option>
                  <option value="Healthcare">{ui.auth.industryHealthcare}</option>
                  <option value="Retail">{ui.auth.industryRetail}</option>
                  <option value="Other">{ui.auth.industryOther}</option>
                </select>
              </div>
              {/* Datos fiscales DIAN — visuales, sin persistencia (spec signup-stytch-compliance) */}
              <div className="rounded-lg border border-dashed border-emerald-500/40 bg-emerald-500/5 p-4 space-y-4">
                <div>
                  <p className="text-sm font-semibold text-emerald-700">
                    {ui.onboarding.wizardCompanyTitle}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {ui.onboarding.wizardCompanySubtitle}
                  </p>
                </div>
                <div>
                  <label className="block text-sm font-medium text-muted-foreground mb-1">
                    {ui.onboarding.wizardNit}
                  </label>
                  <Input
                    type="text"
                    value={visualNit}
                    onChange={(e) => setVisualNit(e.target.value)}
                    disabled={isLoading}
                    placeholder="901.123.456-7"
                    className="border-border bg-background text-foreground placeholder:text-muted-foreground focus:border-primary focus:ring-primary/30"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-muted-foreground mb-1">
                    {ui.onboarding.wizardRegimen}
                  </label>
                  <select
                    className={cn(
                      "w-full rounded-md border border-border bg-background px-3 py-2 text-foreground",
                      "focus:border-primary focus:ring-primary/30 focus:outline-none focus:ring-2"
                    )}
                    value={visualRegimen}
                    onChange={(e) => setVisualRegimen(e.target.value)}
                    disabled={isLoading}
                  >
                    <option value={ui.onboarding.wizardRegimenSimplificado}>
                      {ui.onboarding.wizardRegimenSimplificado}
                    </option>
                    <option value={ui.onboarding.wizardRegimenComun}>
                      {ui.onboarding.wizardRegimenComun}
                    </option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium text-muted-foreground mb-1">
                    {ui.onboarding.wizardCity}
                  </label>
                  <select
                    className={cn(
                      "w-full rounded-md border border-border bg-background px-3 py-2 text-foreground",
                      "focus:border-primary focus:ring-primary/30 focus:outline-none focus:ring-2"
                    )}
                    value={visualCity}
                    onChange={(e) => setVisualCity(e.target.value)}
                    disabled={isLoading}
                  >
                    {ui.onboarding.wizardCities.map((city) => (
                      <option key={city} value={city}>
                        {city}
                      </option>
                    ))}
                  </select>
                </div>
              </div>
              <div className="flex gap-3">
                <Button
                  variant="outline"
                  onClick={goBack}
                  disabled={isLoading}
                  className="flex-1 border-border text-muted-foreground hover:bg-muted hover:text-foreground"
                >
                  {ui.common.back}
                </Button>
                <Button
                  onClick={goNext}
                  disabled={!canContinueOrganization || isLoading}
                  className="flex-1 bg-primary hover:bg-primary/90"
                >
                  {ui.auth.continue} <ArrowRight className="ml-2 h-4 w-4" />
                </Button>
              </div>
            </div>
          )}

          {/* Step 3: Business context */}
          {step === "business" && (
            <div className="space-y-4">
              {/* Paso "Conecta WhatsApp" (plantilla Verifika) */}
              <div>
                <h2 className="text-lg font-semibold text-foreground">
                  {ui.onboarding.wizardConnectWhatsappTitle}
                </h2>
                <p className="text-sm text-muted-foreground">
                  {ui.onboarding.wizardConnectWhatsappSubtitle}
                </p>
              </div>
              <BusinessContextStep
                value={business}
                onChange={updateBusiness}
                onBack={goBack}
                onContinue={sendMagicLink}
                canContinue={canContinueBusiness}
                disabled={isLoading}
                submitLabel={isLoading ? ui.auth.creating : ui.auth.createAccount}
              />
            </div>
          )}

          {!emailSent && (
            <p className="mt-6 text-center text-sm text-muted-foreground">
              {ui.auth.alreadyHaveAccount}{" "}
              <Link href="/auth" className="text-primary hover:underline font-medium">
                {ui.auth.signIn}
              </Link>
            </p>
          )}
        </div>
      </div>
    </div>
  );
}
