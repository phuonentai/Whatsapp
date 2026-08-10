import "@testing-library/jest-dom/vitest";
import { vi } from "vitest";

// jsdom lacks ResizeObserver / matchMedia / scrollIntoView used by radix +
// shadcn components.
class ResizeObserverMock {
  observe() {}
  unobserve() {}
  disconnect() {}
}
if (!("ResizeObserver" in globalThis)) {
  (globalThis as Record<string, unknown>).ResizeObserver = ResizeObserverMock;
}
if (!("scrollIntoView" in window.HTMLElement.prototype)) {
  (window.HTMLElement.prototype as { scrollIntoView?: () => void }).scrollIntoView = () => {};
}
if (!window.matchMedia) {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: (query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: () => {},
      removeListener: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => false,
    }),
  });
}

// Next.js routing + toasts used across client components; mock them at the
// module boundary so component tests stay focused and deterministic.
vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), back: vi.fn(), refresh: vi.fn() }),
  usePathname: () => "/",
  useSearchParams: () => new URLSearchParams(),
  useParams: () => ({}),
  redirect: () => {
    throw new Error("redirect called in test");
  },
}));

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
    warning: vi.fn(),
    info: vi.fn(),
  },
}));
