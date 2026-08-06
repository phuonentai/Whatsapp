import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";
import {
  SESSION_COOKIE_NAME,
  SESSION_JWT_COOKIE_NAME,
} from "@/lib/auth/constants";

// Canonical host for SEO
const CANONICAL_HOST = "yourdomain.com";

// Protected routes that require authentication
const PROTECTED_ROUTES = ["/dashboard", "/settings"];

// Public routes that don't require authentication
const PUBLIC_ROUTES = ["/auth", "/authenticate", "/api/auth"];

export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const url = request.nextUrl.clone();
  const host = request.headers.get("host") || "";

  // Skip redirects for local/dev hosts
  const isLocalhost =
    host.startsWith("localhost") || host.startsWith("127.0.0.1");

  // 1. SEO: Force apex domain (remove www.) - skip for localhost/dev
  if (!isLocalhost && host === `www.${CANONICAL_HOST}`) {
    url.hostname = CANONICAL_HOST;
    return NextResponse.redirect(url, 301);
  }

  // 2. SEO: Enforce HTTPS on canonical domain
  if (!isLocalhost && host === CANONICAL_HOST) {
    const proto = request.headers.get("x-forwarded-proto");
    if (proto && proto !== "https") {
      url.protocol = "https:";
      return NextResponse.redirect(url, 301);
    }
  }

  // 3. Check if route is public (no auth needed)
  const isPublicRoute = PUBLIC_ROUTES.some((route) =>
    pathname.startsWith(route)
  );
  if (isPublicRoute) {
    return NextResponse.next();
  }

  // 4. Check if route is protected (auth required)
  const isProtectedRoute = PROTECTED_ROUTES.some((route) =>
    pathname.startsWith(route)
  );
  if (!isProtectedRoute) {
    return NextResponse.next();
  }

  // 5. Check for session cookies
  const sessionToken = request.cookies.get(SESSION_COOKIE_NAME);
  const sessionJwt = request.cookies.get(SESSION_JWT_COOKIE_NAME);

  // If no session, redirect to login
  if (!sessionToken && !sessionJwt) {
    const loginUrl = new URL("/auth", request.url);
    loginUrl.searchParams.set("returnTo", pathname);
    return NextResponse.redirect(loginUrl);
  }

  // Session exists, allow access
  return NextResponse.next();
}

export const config = {
  matcher: [
    /*
     * Match all request paths except:
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico (favicon file)
     * - public files (images, etc.)
     */
    "/((?!_next/static|_next/image|favicon.ico|icon.png|.*\\.(?:svg|png|jpg|jpeg|gif|webp)$).*)",
  ],
};
