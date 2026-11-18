FROM postgres:16-alpine

ARG APP_VERSION=dev
ENV APP_VERSION=${APP_VERSION}
LABEL org.opencontainers.image.revision="${APP_VERSION}"

# Scripts in this folder run once, on first start with an empty data directory.
COPY init.sql /docker-entrypoint-initdb.d/01-init.sql

EXPOSE 5432

# Credentials come from POSTGRES_USER, POSTGRES_PASSWORD and POSTGRES_DB at runtime.
# Mount a persistent volume at /var/lib/postgresql/data.
HEALTHCHECK --interval=10s --timeout=5s --start-period=20s --retries=5 \
    CMD pg_isready -U "${POSTGRES_USER:-dhaba}" -d "${POSTGRES_DB:-dhaba}" || exit 1
