import { useCallback, useMemo, useRef, useState } from "react";
import { signupRepository } from "@/lib/api/api/repositories/signup-repository";
import {
  SignupBusinessContext,
  SignupOrganization,
  SignupOwner,
  SignupResult,
} from "@/lib/models/signup.model";

export type SignupStep = "account" | "organization" | "business";

interface UseSignupFlowState {
  step: SignupStep;
  owner: SignupOwner;
  organization: SignupOrganization;
  business: SignupBusinessContext;
  isLoading: boolean;
  error: string | null;
  emailSent: boolean;
  result: SignupResult | null;
  stepIndex: number;
  canContinueAccount: boolean;
  canContinueOrganization: boolean;
  canContinueBusiness: boolean;
  goBack: () => void;
  goNext: () => void;
  sendMagicLink: () => Promise<void>;
  updateOwner: (updates: Partial<SignupOwner>) => void;
  updateOrganization: (updates: Partial<SignupOrganization>) => void;
  updateBusiness: (updates: Partial<SignupBusinessContext>) => void;
  reset: () => void;
}

const defaultOwner: SignupOwner = {
  fullName: "",
  email: "",
};

const defaultOrganization: SignupOrganization = {
  displayName: "",
  industry: "Technology",
};

const defaultBusiness: SignupBusinessContext = {
  whatsappReadiness: "planning",
  businessGoal: "",
};

export function useSignupFlow(): UseSignupFlowState {
  const [step, setStep] = useState<SignupStep>("account");
  const [owner, setOwner] = useState<SignupOwner>(defaultOwner);
  const [organization, setOrganization] = useState<SignupOrganization>(
    defaultOrganization
  );
  const [business, setBusiness] = useState<SignupBusinessContext>(defaultBusiness);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [emailSent, setEmailSent] = useState(false);
  const [result, setResult] = useState<SignupResult | null>(null);

  // Ref to prevent duplicate submissions
  const isSubmittingRef = useRef(false);

  const stepIndex = useMemo(() => {
    switch (step) {
      case "account":
        return 0;
      case "organization":
        return 1;
      case "business":
        return 2;
      default:
        return 0;
    }
  }, [step]);

  const canContinueAccount = useMemo(() => {
    return (
      owner.fullName.trim().length >= 2 &&
      /.+@.+\..+/.test(owner.email)
    );
  }, [owner]);

  const canContinueOrganization = useMemo(() => {
    return (
      organization.displayName.trim().length >= 2 &&
      organization.industry.trim().length > 0
    );
  }, [organization]);

  const canContinueBusiness = useMemo(() => {
    return business.businessGoal.trim().length >= 2;
  }, [business]);

  const goBack = useCallback(() => {
    setError(null);
    if (step === "organization") {
      setStep("account");
    } else if (step === "business") {
      setStep("organization");
    }
  }, [step]);

  const goNext = useCallback(() => {
    setError(null);
    if (step === "account" && canContinueAccount) {
      setStep("organization");
    } else if (step === "organization" && canContinueOrganization) {
      setStep("business");
    }
  }, [step, canContinueAccount, canContinueOrganization]);

  const updateOwner = useCallback((updates: Partial<SignupOwner>) => {
    setOwner((prev) => ({ ...prev, ...updates }));
    setError(null); // Clear error when user types
  }, []);

  const updateOrganization = useCallback((updates: Partial<SignupOrganization>) => {
    setOrganization((prev) => ({ ...prev, ...updates }));
    setError(null); // Clear error when user types
  }, []);

  const updateBusiness = useCallback((updates: Partial<SignupBusinessContext>) => {
    setBusiness((prev) => ({ ...prev, ...updates }));
    setError(null); // Clear error when user types
  }, []);

  const reset = useCallback(() => {
    setOwner(defaultOwner);
    setOrganization(defaultOrganization);
    setBusiness(defaultBusiness);
    setStep("account");
    setError(null);
    setEmailSent(false);
    setResult(null);
  }, []);

  const sendMagicLink = useCallback(async () => {
    // Prevent duplicate submissions
    if (isSubmittingRef.current) return;

    if (!canContinueOrganization) {
      setError("Please fill in all required fields correctly");
      return;
    }

    isSubmittingRef.current = true;
    setIsLoading(true);
    setError(null);

    try {
      // Backend signup endpoint already sends magic link via Stytch
      // No need to call sendMagicLink() Server Action separately
      const signupResult = await signupRepository.createOrganizationWithMagicLink(
        owner,
        organization
      );

      setResult(signupResult);
      setEmailSent(true);
    } catch (signupError) {
      const message =
        signupError instanceof Error
          ? signupError.message
          : "Failed to create account. Please try again.";
      setError(message);
      setEmailSent(false);
    } finally {
      setIsLoading(false);
      isSubmittingRef.current = false;
    }
  }, [owner, organization, canContinueOrganization]);

  return {
    step,
    owner,
    organization,
    business,
    isLoading,
    error,
    emailSent,
    result,
    stepIndex,
    canContinueAccount,
    canContinueOrganization,
    canContinueBusiness,
    goBack,
    goNext,
    sendMagicLink,
    updateOwner,
    updateOrganization,
    updateBusiness,
    reset,
  };
}
