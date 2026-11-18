# ---- Build stage ----
# Creates a stock Laravel 13 app without dev packages, adds predis, then copies our code
# over it (overlay/). Composer and the build tools stay in this stage.
FROM composer:2 AS builder

WORKDIR /app
RUN composer create-project laravel/laravel:^13.0 . \
        --no-dev --prefer-dist --no-interaction --no-progress --ignore-platform-reqs && \
    composer require predis/predis:^2.2 \
        --update-no-dev --no-interaction --no-progress --ignore-platform-reqs && \
    rm -f .env database/database.sqlite

COPY overlay/ ./
RUN rm -rf tests && \
    composer dump-autoload --optimize --classmap-authoritative --no-dev --no-interaction

# ---- Final stage: PHP-FPM + Nginx, production settings ----
FROM php:8.3-fpm-alpine

ARG APP_VERSION=dev
ENV APP_VERSION=${APP_VERSION} \
    APP_ENV=production \
    APP_DEBUG=false \
    LOG_CHANNEL=stderr \
    SESSION_DRIVER=array \
    CACHE_STORE=array \
    DB_CONNECTION=sqlite \
    REDIS_CLIENT=predis \
    REDIS_HOST=redis \
    REDIS_PORT=6379
LABEL org.opencontainers.image.revision="${APP_VERSION}"

RUN apk add --no-cache nginx && \
    docker-php-ext-install opcache && \
    cp "$PHP_INI_DIR/php.ini-production" "$PHP_INI_DIR/php.ini" && \
    rm -f /usr/local/etc/php-fpm.d/*.conf && \
    addgroup -S app && adduser -S -G app app

COPY opcache.ini  $PHP_INI_DIR/conf.d/zz-opcache.ini
COPY fpm-pool.conf /usr/local/etc/php-fpm.d/app.conf
COPY nginx.conf   /etc/nginx/nginx.conf

WORKDIR /var/www
COPY --from=builder --chown=app:app /app .
COPY --chown=app:app docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh && chown -R app:app storage bootstrap/cache

USER app
EXPOSE 9000

# /health checks Redis and returns {"status":"ok","version":"<commit>"}
HEALTHCHECK --interval=15s --timeout=5s --start-period=20s --retries=3 \
    CMD wget -qO- http://127.0.0.1:9000/health || exit 1

ENTRYPOINT ["/docker-entrypoint.sh"]
