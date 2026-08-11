"use client";

import { useSignupFlow } from "@/hooks/use-signup-flow";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ArrowRight, Home, Inbox } from "lucide-react";
import { PRODUCT_NAME } from "@/lib/brand";
import { ui } from "@/lib/copy/ui";
import { BusinessContextStep } from "@/components/onboarding/business-context-step";
import Link from "next/link";

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

  // Success view after email sent
  if (emailSent) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-muted/40 px-6">
        <div className="w-full max-w-md text-center space-y-6">
          <div className="mx-auto h-14 w-14 bg-primary/10 rounded-full flex items-center justify-center">
            <Inbox className="h-7 w-7 text-primary" />
          </div>
          <h1 className="text-2xl font-semibold text-foreground">{ui.auth.emailSentTitle}</h1>
          <p className="text-muted-foreground">
            {ui.auth.emailSentIntro} <strong>{owner.email}</strong>. {ui.auth.emailSentCta}
          </p>
          <Link href="/auth">
            <Button variant="outline">{ui.auth.backToSignIn}</Button>
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-muted/40 px-6">
      <div className="w-full max-w-md">
        <div className="bg-card p-8 rounded-2xl shadow-lg border border-border">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h1 className="text-2xl font-semibold text-foreground">
                {ui.auth.title}
              </h1>
              <p className="text-sm text-muted-foreground mt-1">
                {ui.auth.subtitle} {PRODUCT_NAME}
              </p>
            </div>
            <Link href="/" className="flex items-center gap-1.5 text-xs text-muted-foreground hover:text-primary transition-colors">
              <Home className="h-3.5 w-3.5" />
              <span>{ui.auth.home}</span>
            </Link>
          </div>

          {error && (
            <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-md">
              <p className="text-sm text-red-800">{error}</p>
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
                />
              </div>
              <Button
                onClick={goNext}
                disabled={!canContinueAccount || isLoading}
                className="w-full"
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
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-muted-foreground mb-1">
                  {ui.auth.industry}
                </label>
                <select
                  className="w-full px-3 py-2 border border-gray-300 rounded-md"
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
              <div className="flex gap-3">
                <Button
                  variant="outline"
                  onClick={goBack}
                  disabled={isLoading}
                  className="flex-1"
                >
                  {ui.common.back}
                </Button>
                <Button
                  onClick={goNext}
                  disabled={!canContinueOrganization || isLoading}
                  className="flex-1"
                >
                  {ui.auth.continue} <ArrowRight className="ml-2 h-4 w-4" />
                </Button>
              </div>
            </div>
          )}

          {/* Step 3: Business context */}
          {step === "business" && (
            <BusinessContextStep
              value={business}
              onChange={updateBusiness}
              onBack={goBack}
              onContinue={sendMagicLink}
              canContinue={canContinueBusiness}
              disabled={isLoading}
              submitLabel={isLoading ? ui.auth.creating : ui.auth.createAccount}
            />
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
