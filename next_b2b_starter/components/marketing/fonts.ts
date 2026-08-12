import { Inter, Sora } from "next/font/google";

/**
 * Typography for the Verifika/ChatFlow identity (adopt-verifika-template):
 * Sora (display) via --font-heading + Inter (body) via --font-sans.
 * Both variables are set on <html> by next/font; tailwind.config.ts maps
 * font-heading -> var(--font-heading) and sans -> var(--font-sans).
 */
export const inter = Inter({
  subsets: ["latin"],
  display: "swap",
  variable: "--font-sans",
});

export const sora = Sora({
  subsets: ["latin"],
  display: "swap",
  variable: "--font-heading",
});
