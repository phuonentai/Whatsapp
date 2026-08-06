"use client";

import { createContext, useContext } from "react";
import type { StytchClientConfig } from "@/lib/auth/config-types";

const StytchConfigContext = createContext<StytchClientConfig | null>(null);

export function StytchConfigProvider({
  children,
  config,
}: {
  children: React.ReactNode;
  config: StytchClientConfig;
}) {
  return (
    <StytchConfigContext.Provider value={config}>
      {children}
    </StytchConfigContext.Provider>
  );
}

export function useStytchConfig(): StytchClientConfig {
  const config = useContext(StytchConfigContext);
  if (!config) {
    throw new Error("useStytchConfig must be used within StytchConfigProvider");
  }
  return config;
}
