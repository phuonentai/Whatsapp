# =============================================================================
# Frontend - Production Dockerfile
# =============================================================================
# Multi-stage build for optimized Next.js 15 deployment with SSR support
# Final image size: ~120-150MB (vs ~500MB+ without optimization)
# Node.js 22 LTS (Jod) with Alpine Linux 3.22
# =============================================================================

# -----------------------------------------------------------------------------
# Stage 1: Dependencies Installation
# -----------------------------------------------------------------------------
FROM node:22-alpine3.22 AS deps

# Install necessary system dependencies
RUN apk add --no-cache libc6-compat

WORKDIR /app

# Install pnpm globally
RUN corepack enable && corepack prepare pnpm@latest --activate

# Copy package management files
COPY package.json pnpm-lock.yaml .npmrc* ./

# Install dependencies using pnpm
# .npmrc ensures sharp is built correctly for image optimization
RUN pnpm install --frozen-lockfile --prod=false

# -----------------------------------------------------------------------------
# Stage 2: Application Builder
# -----------------------------------------------------------------------------
FROM node:22-alpine3.22 AS builder

WORKDIR /app

# Install pnpm
RUN corepack enable && corepack prepare pnpm@latest --activate

# Copy dependencies from deps stage
COPY --from=deps /app/node_modules ./node_modules

# Copy application source
COPY . .

# Build arguments for NEXT_PUBLIC_* environment variables
# These must be set at build time for client-side code
ARG NEXT_PUBLIC_APP_BASE_URL
ARG NEXT_PUBLIC_API_BASE_URL
ARG NEXT_PUBLIC_STYTCH_PUBLIC_TOKEN
ARG NEXT_PUBLIC_STYTCH_PROJECT_ID
ARG NEXT_PUBLIC_STYTCH_PROJECT_ENV
ARG NEXT_PUBLIC_POLAR_API_SANDBOX_ACCESS_TOKEN
ARG NEXT_PUBLIC_POLAR_API_PRODUCTION_ACCESS_TOKEN
ARG NEXT_PUBLIC_UMAMI_WEBSITE_ID
ARG NEXT_PUBLIC_UMAMI_SCRIPT_URL

# Set build-time environment variables
ENV NEXT_PUBLIC_APP_BASE_URL=$NEXT_PUBLIC_APP_BASE_URL
ENV NEXT_PUBLIC_API_BASE_URL=$NEXT_PUBLIC_API_BASE_URL
ENV NEXT_PUBLIC_STYTCH_PUBLIC_TOKEN=$NEXT_PUBLIC_STYTCH_PUBLIC_TOKEN
ENV NEXT_PUBLIC_STYTCH_PROJECT_ID=$NEXT_PUBLIC_STYTCH_PROJECT_ID
ENV NEXT_PUBLIC_STYTCH_PROJECT_ENV=$NEXT_PUBLIC_STYTCH_PROJECT_ENV
ENV NEXT_PUBLIC_POLAR_API_SANDBOX_ACCESS_TOKEN=$NEXT_PUBLIC_POLAR_API_SANDBOX_ACCESS_TOKEN
ENV NEXT_PUBLIC_POLAR_API_PRODUCTION_ACCESS_TOKEN=$NEXT_PUBLIC_POLAR_API_PRODUCTION_ACCESS_TOKEN
ENV NEXT_PUBLIC_UMAMI_WEBSITE_ID=$NEXT_PUBLIC_UMAMI_WEBSITE_ID
ENV NEXT_PUBLIC_UMAMI_SCRIPT_URL=$NEXT_PUBLIC_UMAMI_SCRIPT_URL

# Disable Next.js telemetry
ENV NEXT_TELEMETRY_DISABLED=1

# Build the application
# next.config.ts already has output: "standalone" configured
RUN pnpm build

# -----------------------------------------------------------------------------
# Stage 3: Production Runtime
# -----------------------------------------------------------------------------
FROM node:22-alpine3.22 AS runner

WORKDIR /app

# Set production environment
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

# Create non-root user for security
RUN addgroup --system --gid 1001 nodejs && \
    adduser --system --uid 1001 nextjs

# Copy public assets
COPY --from=builder /app/public ./public

# Copy standalone build output (optimized by Next.js)
# This includes only necessary dependencies and files
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

# Switch to non-root user
USER nextjs

# Expose application port
EXPOSE 3000

# Set runtime environment variables
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

# Health check for container orchestration
HEALTHCHECK --interval=30s --timeout=3s --start-period=40s --retries=3 \
  CMD node -e "require('http').get('http://localhost:3000/api/health', (r) => {process.exit(r.statusCode === 200 ? 0 : 1)})"

# Start the application
# Use node directly (not npm/pnpm) for proper signal handling and graceful shutdown
CMD ["node", "server.js"]
