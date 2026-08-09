"use client";

import { useMemo } from "react";
import { StytchB2BProvider } from "@stytch/nextjs/b2b";
import { createStytchB2BUIClient } from "@stytch/nextjs/b2b/ui";
import {
  SESSION_COOKIE_NAME,
  SESSION_JWT_COOKIE_NAME,
} from "@/lib/auth/constants";
import { StytchConfigProvider } from "@/lib/contexts/stytch-config-context";
import type { StytchClientConfig } from "@/lib/auth/config-types";

interface Props {
  children: React.ReactNode;
  config: StytchClientConfig;
}

const COOKIE_OPTIONS = {
  opaqueTokenCookieName: SESSION_COOKIE_NAME,
  jwtCookieName: SESSION_JWT_COOKIE_NAME,
  path: "/",
  availableToSubdomains: false,
  domain: "",
} as const;

export function StytchProvider({ children, config }: Props) {
  // Mock auth (E2E only): the Stytch SDK cannot validate mock sessions, so
  // skip the provider entirely. Consumers already handle a missing provider.
  const mockAuth = process.env.NEXT_PUBLIC_AUTH_MOCK_ENABLED === "true";

  const stytchClient = useMemo(() => {
    if (mockAuth) {
      return null;
    }

    if (!config.publicToken) {
      throw new Error(
        "NEXT_PUBLIC_STYTCH_PUBLIC_TOKEN is required to initialize Stytch."
      );
    }

    return createStytchB2BUIClient(config.publicToken, {
      cookieOptions: COOKIE_OPTIONS,
    });
  }, [config.publicToken, mockAuth]);

  if (mockAuth) {
    return <StytchConfigProvider config={config}>{children}</StytchConfigProvider>;
  }

  return (
    <StytchB2BProvider stytch={stytchClient}>
      <StytchConfigProvider config={config}>
        {children}
      </StytchConfigProvider>
    </StytchB2BProvider>
  );
}
