# Dhaba Orders

Seven services, each in its own folder with its own Dockerfile. Build each into an image, then deploy each as a Kubernetes Service.

| Folder | Kind | Stack | Port | Job |
|---|---|---|---|---|
| `web` | Stateless | React (Vite) on Nginx | 8080 | Order page, kitchen screen, stats panel; proxies `/api/*` |
| `menu-api` | Stateless | Python, FastAPI | 8000 | Menu from Postgres, cached in Redis |
| `orders-api` | Stateless | Go | 8080 | Saves orders, queues jobs, writes stats. `/info` shows version and pod |
| `kitchen-worker` | Stateless, no web port | Java, Spring Boot | none | Takes queue jobs, moves orders new, cooking, ready |
| `stats-api` | Stateless | PHP, Laravel 11 | 9000 | Orders per minute and top dishes from Redis |
| `postgres` | Stateful | PostgreSQL 16 | 5432 | Orders and menu. Volume at `/var/lib/postgresql/data` |
| `redis` | Stateful | Redis 7 | 6379 | Cache and job queue. Volume at `/data` |

## Data flow

1. `web` calls `/api/menu`. Nginx sends it to `menu-api`, which reads Redis first, then Postgres.
2. `web` posts to `/api/orders`. `orders-api` saves the order and its items in one transaction, then does `LPUSH orders:queue {"order_id": N}` and updates `stats:orders_ts` and `stats:top_dishes`.
3. `kitchen-worker` pops `orders:queue`, sets the order to `cooking`, waits 5 seconds, sets it to `ready`.
4. The kitchen tab polls `/api/orders` every 5 seconds. The stats tab polls `/api/stats`.

## Build the images and push them to ECR

See [docs/BUILD-AND-PUSH.md](docs/BUILD-AND-PUSH.md). Short version:

    scripts/create-ecr-repos.sh      # once
    scripts/build-and-push.sh        # build, tag with the commit, push

## Environment variables

| Service | Variable | Default |
|---|---|---|
| menu-api | `DATABASE_URL`, `REDIS_URL`, `CACHE_TTL_SECONDS`, `APP_VERSION` | `postgres://dhaba:dhaba@postgres:5432/dhaba`, `redis://redis:6379`, 300 |
| orders-api | `DATABASE_URL` (needs `?sslmode=disable` in-cluster), `REDIS_URL`, `PORT`, `POD_NAME`, `APP_VERSION` | as above, 8080 |
| kitchen-worker | `DB_URL` (JDBC form), `DB_USER`, `DB_PASSWORD`, `REDIS_URL`, `COOK_MILLIS` | `jdbc:postgresql://postgres:5432/dhaba`, dhaba, dhaba, 5000 |
| stats-api | `REDIS_HOST`, `REDIS_PORT`, `APP_KEY` (optional), `APP_VERSION` | `redis`, 6379 |
| postgres | `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` | none, set them |

Kubernetes notes for the next step:

- Keep the Service names `menu-api`, `orders-api`, `stats-api`, `postgres`, `redis`. Nginx and the defaults rely on them.
- In `orders-api`, set `POD_NAME` from the downward API (`metadata.name`) so `/info` shows the pod.
- `kitchen-worker` has no port. For a liveness probe use an exec probe: `test -n "$(find /tmp/kitchen-heartbeat -mmin -1)"`.
- Use a StatefulSet (or a Deployment with a PVC) for `postgres` and `redis`.

## Changes from the first draft

- `KitchenScreen` used the status `pending`; the database uses `new`. Fixed.
- `orders-api`: `-X main.version` cannot override a variable set from `os.Getenv`; fixed. The `--health-check` flag the Dockerfile used did not exist; added. Name and price now come from the menu table, not the client. Stats keys the stats-api reads are now written by orders-api.
- `redis.conf`: `allkeys-lru` could evict queued jobs. Changed to `noeviction`.
- `kitchen-worker`: `DATABASE_URL` conflicted with the `postgres://` form the other services use, so it now reads `DB_URL`. The Docker health check pointed at an actuator that does not exist; it now uses a heartbeat file. Constructor injection makes it testable without a Spring context. Checkstyle removed from the build because it can fail the image build on style warnings.
- `stats-api`: `config:cache` makes `env()` return null at runtime; replaced with `getenv`. The image now builds a stock Laravel 11 app (no dev packages) and applies `overlay/` on top, so no framework boilerplate is committed. It runs PHP-FPM and Nginx as a non-root user, and caches config and routes when the container starts.
- `web`: runs as a non-root Nginx on port 8080. The API hosts are resolved per request, so the container starts without the backends.
- Every image carries the commit as `APP_VERSION` and as an image label. `docker-compose.yml` was removed; images are built and pushed with the scripts in `scripts/`.

## Testing status

Run here: `web` (3 tests pass, production build succeeds), `menu-api` (4 tests pass).
Not run here, because the sandbox has no Docker daemon and cannot download Go, Maven or Composer dependencies: `orders-api`, `kitchen-worker`, `stats-api`, and all eight Docker builds. The scripts pass `bash -n`; they have not run against AWS. `orders-api/main.go` is gofmt-clean and the PHP files pass `php -l`. Run `go mod tidy` in `orders-api` once to create `go.sum`, then commit it.
