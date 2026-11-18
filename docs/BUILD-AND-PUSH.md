# Build the images and push them to ECR

No Docker Compose. Each service is built from its own Dockerfile, tagged with the Git commit, and pushed to its own ECR repository.

| Image | Build context | Listens on | Runs as |
|---|---|---|---|
| web | `web/` | 8080 | nginx (101) |
| menu-api | `menu-api/` | 8000 | app |
| orders-api | `orders-api/` | 8080 | 65534 |
| kitchen-worker | `kitchen-worker/` | none | app |
| stats-api | `stats-api/` | 9000 | app |
| postgres | `postgres/` | 5432 | image default |
| redis | `redis/` | 6379 | image default |
| ops-monitor | repository root (`-f ops-monitor/Dockerfile .`) | none | app |

## 1. Prerequisites

- Docker running on your laptop
- AWS CLI v2, signed in to your team's AWS account from your laptop with the access your trainer gave you (`aws sts get-caller-identity` must work). The OIDC role is for GitHub Actions only; it does not work from a laptop.
- You are inside your Git clone, on the commit you want to build

## 2. Set the variables

    export TEAM=yourteam
    export AWS_ACCOUNT=123456789012
    export AWS_REGION=eu-west-1
    export REGISTRY=$AWS_ACCOUNT.dkr.ecr.$AWS_REGION.amazonaws.com
    export TAG=$(git rev-parse HEAD)

The tag is the commit SHA. Never push or deploy `latest`.

## 3. Create the ECR repositories (once)

    scripts/create-ecr-repos.sh

This creates `dhaba-TEAM/web`, `dhaba-TEAM/menu-api` and the other six, with scan on push and a rule that keeps the newest 20 images. A second run prints "already exists" and changes nothing.

## 4. Build one image by hand

Start with `web` to see what the script does for every service:

    docker build \
      --platform linux/amd64 \
      --build-arg APP_VERSION=$TAG \
      -t $REGISTRY/dhaba-$TEAM/web:$TAG \
      ./web

- `--platform linux/amd64` is needed on Apple Silicon and other ARM laptops. The cluster nodes run amd64, and an arm64 image fails there with `exec format error`.
- `--build-arg APP_VERSION=$TAG` puts the commit into the image, so the service can show it.

Same command for the others; only the last argument and the repository name change:

    docker build --platform linux/amd64 --build-arg APP_VERSION=$TAG -t $REGISTRY/dhaba-$TEAM/menu-api:$TAG        ./menu-api
    docker build --platform linux/amd64 --build-arg APP_VERSION=$TAG -t $REGISTRY/dhaba-$TEAM/orders-api:$TAG      ./orders-api
    docker build --platform linux/amd64 --build-arg APP_VERSION=$TAG -t $REGISTRY/dhaba-$TEAM/kitchen-worker:$TAG  ./kitchen-worker
    docker build --platform linux/amd64 --build-arg APP_VERSION=$TAG -t $REGISTRY/dhaba-$TEAM/stats-api:$TAG       ./stats-api
    docker build --platform linux/amd64 --build-arg APP_VERSION=$TAG -t $REGISTRY/dhaba-$TEAM/postgres:$TAG        ./postgres
    docker build --platform linux/amd64 --build-arg APP_VERSION=$TAG -t $REGISTRY/dhaba-$TEAM/redis:$TAG           ./redis

`ops-monitor` is built from the repository root, because it copies `scripts/monitor/` (Phase 5):

    docker build --platform linux/amd64 --build-arg APP_VERSION=$TAG \
      -t $REGISTRY/dhaba-$TEAM/ops-monitor:$TAG -f ops-monitor/Dockerfile .

## 5. Check the images before pushing

Non-root user (an empty result means root; only postgres and redis may show that):

    docker image inspect --format '{{.Config.User}}' $REGISTRY/dhaba-$TEAM/web:$TAG

Sizes, for your pull request:

    docker images --format 'table {{.Repository}}\t{{.Tag}}\t{{.Size}}' | grep dhaba-$TEAM

Run one image on its own. `web` starts without the backends:

    docker run --rm -p 8080:8080 $REGISTRY/dhaba-$TEAM/web:$TAG
    # in another terminal
    curl localhost:8080/healthz
    curl localhost:8080/version.txt        # prints the commit

## 6. Log in to ECR and push

    aws ecr get-login-password --region $AWS_REGION \
      | docker login --username AWS --password-stdin $REGISTRY

    docker push $REGISTRY/dhaba-$TEAM/web:$TAG
    # repeat for the other services

The login lasts 12 hours.

## 7. Verify in ECR

    aws ecr describe-images --region $AWS_REGION \
      --repository-name dhaba-$TEAM/web --query 'imageDetails[].imageTags'

After the scan finishes (a minute or two):

    aws ecr describe-image-scan-findings --region $AWS_REGION \
      --repository-name dhaba-$TEAM/web --image-id imageTag=$TAG \
      --query 'imageScanFindings.findingSeverityCounts'

Any `CRITICAL` count above zero fails the pipeline in Phase 3, so fix it now.

## The same thing with one script

    scripts/build-and-push.sh                 # all services
    scripts/build-and-push.sh web stats-api   # only these
    PUSH=0 scripts/build-and-push.sh          # build and show sizes, no push

It logs in to ECR, builds with the commit as tag, pushes, and prints a table of sizes and users to paste into the pull request. `ops-monitor` is skipped until `scripts/monitor/` exists.

## Problems you may hit

| Message | Cause and fix |
|---|---|
| `exec format error` when the pod starts | Image was built for arm64. Rebuild with `--platform linux/amd64` |
| `no basic auth credentials` on push | ECR login expired or was for another region. Repeat step 6 |
| `name unknown: The repository ... does not exist` | Run `scripts/create-ecr-repos.sh` first |
| `go.sum` missing in `orders-api` | Run `go mod tidy` in `orders-api/` once and commit `go.sum` |
| Pods say `ErrImagePull` | The cluster cannot pull from ECR yet. That is the ECR credential choice in Phase 3 (C9) |
