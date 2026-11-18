#!/bin/sh
set -e

# Laravel needs an APP_KEY. Pass one in (Kubernetes Secret) so all replicas share it;
# otherwise make one for this container. Nothing in this service uses encryption or sessions.
if [ -z "$APP_KEY" ]; then
    APP_KEY="$(php artisan key:generate --show)"
    export APP_KEY
fi

# Cache config and routes now, at start, so they use the real environment of this container
# (REDIS_HOST and so on), not the values from build time.
php artisan config:cache
php artisan route:cache

mkdir -p /tmp/nginx
php-fpm &

# Nginx runs in the foreground as the main process. If PHP-FPM dies, /health fails
# and the health check or Kubernetes probe restarts the container.
exec nginx -e /dev/stderr -g 'daemon off;'
