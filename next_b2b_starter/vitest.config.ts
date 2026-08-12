import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import path from "path";

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: "jsdom",
    setupFiles: ["./test/setup.ts"],
    include: ["**/*.test.{ts,tsx}"],
    exclude: ["node_modules/**", "e2e/**", ".next/**", "test-results/**"],
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "."),
      // Next.js uses this package to guard server-only modules from client
      // bundles; vitest resolves it to a no-op so server auth code is testable.
      "server-only": path.resolve(__dirname, "test/mocks/server-only.ts"),
    },
  },
});
