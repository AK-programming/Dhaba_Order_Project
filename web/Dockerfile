# ---- Build stage: Node.js lives here only ----
FROM node:22-alpine AS builder

WORKDIR /app
COPY package.json package-lock.json* ./
RUN if [ -f package-lock.json ]; then npm ci; else npm install; fi

COPY . .
ARG APP_VERSION=dev
# version.txt lets you check which commit is running: curl http://HOST/version.txt
RUN npm run build && echo "${APP_VERSION}" > dist/version.txt

# ---- Final stage: Nginx only, runs as user 101 (non-root), listens on 8080 ----
FROM nginxinc/nginx-unprivileged:1.27-alpine

USER root
RUN apk upgrade --no-cache
USER 101

ARG APP_VERSION=dev
ENV APP_VERSION=${APP_VERSION}
LABEL org.opencontainers.image.revision="${APP_VERSION}"

# The entrypoint fills in ${NGINX_LOCAL_RESOLVERS} (the DNS server of the container)
# and writes /etc/nginx/conf.d/default.conf when the container starts.
COPY nginx.conf /etc/nginx/templates/default.conf.template
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1
