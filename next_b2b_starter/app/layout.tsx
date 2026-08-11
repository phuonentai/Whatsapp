import "./globals.css";
import type { Metadata, Viewport } from "next";
import { Toaster } from "sonner";
import { ThemeProvider } from "next-themes";
import { AuthProvider } from "@/lib/contexts/auth-context";
import { StytchProvider } from "@/components/auth/stytch-provider";
import { authBootstrap } from "@/lib/auth/bootstrap";
import { buildStytchClientConfig } from "@/lib/auth/stytch-server";
import { PRODUCT_NAME } from "@/lib/brand";
import { QueryProvider } from "@/lib/providers/query-provider";
import { JsonLd } from "@/components/seo/jsonld";
import { Inter } from "next/font/google";
import Script from "next/script";

const inter = Inter({
  subsets: ["latin"],
  display: "swap",
  variable: "--font-sans",
});

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  maximumScale: 5,
  themeColor: "#000000",
};

export const metadata: Metadata = {
  metadataBase: new URL("https://yourdomain.com"),
  icons: {
    icon: "/icon.png",
  },
  title: {
    default: `${PRODUCT_NAME} | Next.js Starter with Auth & Billing`,
    template: `%s | ${PRODUCT_NAME}`,
  },
  description:
    "A modern Next.js starter template with authentication, billing, and team management built in. Perfect for launching your SaaS application quickly.",
  keywords: [
    "nextjs starter",
    "saas starter",
    "authentication",
    "billing integration",
    "team management",
    "nextjs template",
    "react starter",
    "typescript starter",
    "tailwind starter",
  ],
  authors: [{ name: "Your Team" }],
  creator: PRODUCT_NAME,
  publisher: PRODUCT_NAME,
  formatDetection: {
    email: false,
    address: false,
    telephone: false,
  },
  alternates: {
    canonical: "/",
    languages: {
      "en-US": "/",
    },
  },
  openGraph: {
    type: "website",
    url: "https://yourdomain.com/",
    siteName: PRODUCT_NAME,
    title: `${PRODUCT_NAME} | Next.js Starter with Auth & Billing`,
    description:
      "A modern Next.js starter template with authentication, billing, and team management built in. Perfect for launching your SaaS application quickly.",
    locale: "en_US",
  },
  twitter: {
    card: "summary_large_image",
    title: `${PRODUCT_NAME} | Next.js Starter with Auth & Billing`,
    description:
      "A modern Next.js starter template with authentication, billing, and team management built in. Perfect for launching your SaaS application quickly.",
  },
  robots: {
    index: true,
    follow: true,
    nocache: false,
    googleBot: {
      index: true,
      follow: true,
      "max-image-preview": "large",
      "max-snippet": -1,
      "max-video-preview": -1,
    },
  },
  category: "Business",
};

export default async function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const bootstrap = await authBootstrap();
  const stytchConfig = buildStytchClientConfig();

  return (
    <html lang="en" className={inter.variable} suppressHydrationWarning>
      <head>
        <link rel="dns-prefetch" href="https://fonts.googleapis.com" />
        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link rel="preconnect" href="https://fonts.gstatic.com" crossOrigin="anonymous" />
        <link rel="dns-prefetch" href="https://www.youtube.com" />
        <link rel="preconnect" href="https://www.youtube.com" />
        <link rel="dns-prefetch" href="https://img.youtube.com" />
        <link rel="preconnect" href="https://img.youtube.com" />
      </head>
      <body className="antialiased font-sans" suppressHydrationWarning>
        {/* Global JSON-LD: Organization + WebSite */}
        <JsonLd
          id="org-jsonld"
          data={{
            "@context": "https://schema.org",
            "@type": "Organization",
            name: PRODUCT_NAME,
            url: "https://yourdomain.com",
            logo: "https://yourdomain.com/icon.png",
            description: "A modern Next.js starter template with authentication, billing, and team management built in.",
          }}
        />
        <JsonLd
          id="website-jsonld"
          data={{
            "@context": "https://schema.org",
            "@type": "WebSite",
            name: PRODUCT_NAME,
            url: "https://yourdomain.com",
            inLanguage: "en",
          }}
        />
        <StytchProvider config={stytchConfig}>
          <ThemeProvider
            attribute="class"
            defaultTheme="system"
            enableSystem
            disableTransitionOnChange
          >
            <AuthProvider
              initialProfile={bootstrap.profile}
              initialRoles={bootstrap.roles}
              initialPermissions={bootstrap.permissions}
              shouldClearCache={bootstrap.shouldClearCache}
            >
              <QueryProvider>
                {children}
                <Toaster position="top-right" richColors />
              </QueryProvider>
            </AuthProvider>
          </ThemeProvider>
        </StytchProvider>
      </body>
    </html>
  );
}
