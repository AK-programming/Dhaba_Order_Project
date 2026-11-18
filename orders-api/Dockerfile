# ---- Build stage ----
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache ca-certificates
WORKDIR /src
COPY . .

ARG APP_VERSION=dev
# go mod tidy writes go.sum on the first build; commit go.sum afterwards for reproducible builds.
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.version=${APP_VERSION}" -o /out/orders-api .

# ---- Final stage: small Alpine runtime with a supported ECR scan database ----
FROM alpine:3.22

ARG APP_VERSION=dev
ENV APP_VERSION=${APP_VERSION}
LABEL org.opencontainers.image.revision="${APP_VERSION}"

RUN apk upgrade --no-cache && \
    apk add --no-cache ca-certificates

COPY --from=builder /out/orders-api /orders-api

USER 65534:65534
EXPOSE 8080

# scratch has no shell or curl, so the binary checks itself. /info shows the version.
HEALTHCHECK --interval=15s --timeout=5s --start-period=15s --retries=3 \
    CMD ["/orders-api", "--health-check"]

ENTRYPOINT ["/orders-api"]
