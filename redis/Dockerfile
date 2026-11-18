FROM redis:7-alpine

USER root
RUN apk upgrade --no-cache
USER redis

ARG APP_VERSION=dev
ENV APP_VERSION=${APP_VERSION}
LABEL org.opencontainers.image.revision="${APP_VERSION}"

COPY redis.conf /usr/local/etc/redis/redis.conf

EXPOSE 6379

# Mount a persistent volume at /data (appendonly file and snapshots are written there).
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=5 \
    CMD redis-cli ping | grep -q PONG || exit 1

CMD ["redis-server", "/usr/local/etc/redis/redis.conf"]
