# syntax=docker/dockerfile:1.7
# ============================================================
# KALENGO Web (React + Vite + TS) — §10.1, §15
#   dev     : Vite dev server (docker compose)
#   build   : bundle produksi
#   runtime : Nginx dengan header keamanan + cache immutable aset ber-hash
# ============================================================

FROM node:24-alpine AS base
WORKDIR /src
COPY apps/web/package.json apps/web/package-lock.json* ./

# ---------- dev ----------
FROM base AS dev
RUN npm install --no-audit --no-fund
EXPOSE 5173
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0", "--port", "5173"]

# ---------- build ----------
FROM base AS build
RUN npm ci --no-audit --no-fund
COPY apps/web/ ./
RUN npm run build

# ---------- runtime ----------
FROM nginx:1.27-alpine AS runtime
COPY docker/web/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/dist /usr/share/nginx/html
EXPOSE 80
HEALTHCHECK --interval=15s --timeout=3s --retries=5 \
  CMD wget -qO- http://localhost/ >/dev/null || exit 1
