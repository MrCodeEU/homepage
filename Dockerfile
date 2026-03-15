# ============================================
# Stage 1: Build SvelteKit (data baked in at build time)
# ============================================
FROM node:20-alpine AS frontend-builder

WORKDIR /build/frontend

# Install dependencies
COPY frontend/package*.json ./
RUN npm ci

# Make generated data available to the SvelteKit build.
# +page.server.ts reads from process.cwd()/../backend/data/generated/
COPY backend/data/generated/ /build/backend/data/generated/

# Copy frontend source and build (data is embedded into the prerendered HTML)
COPY frontend/ ./
RUN npm run build

# ============================================
# Stage 2: Serve with Caddy (native Brotli + simpler config)
# ============================================
FROM docker.io/library/caddy:2-alpine

COPY --from=frontend-builder /build/frontend/build /srv
COPY Caddyfile /etc/caddy/Caddyfile

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost/ || exit 1
