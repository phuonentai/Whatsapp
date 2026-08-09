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
  const stytchClient = useMemo(() => {
    if (!config.publicToken) {
      throw new Error(
        "NEXT_PUBLIC_STYTCH_PUBLIC_TOKEN is required to initialize Stytch."
      );
    }

    return createStytchB2BUIClient(config.publicToken, {
      cookieOptions: COOKIE_OPTIONS,
    });
  }, [config.publicToken]);

  return (
    <StytchB2BProvider stytch={stytchClient}>
      <StytchConfigProvider config={config}>
        {children}
      </StytchConfigProvider>
    </StytchB2BProvider>
  );
}
