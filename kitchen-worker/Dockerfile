# ---- Build stage (Maven + JDK) ----
FROM maven:3.9-eclipse-temurin-21-alpine AS builder

WORKDIR /build
COPY pom.xml .
RUN mvn -q dependency:go-offline

COPY src ./src
RUN mvn -q package -DskipTests

# ---- Final stage (JRE only, no JDK, no Maven) ----
FROM eclipse-temurin:21-jre-alpine

ARG APP_VERSION=dev
ENV APP_VERSION=${APP_VERSION}
LABEL org.opencontainers.image.revision="${APP_VERSION}"

RUN addgroup -S app && adduser -S -G app app

WORKDIR /app
COPY --from=builder /build/target/kitchen-worker-*.jar app.jar

USER app

# No web port. The worker rewrites /tmp/kitchen-heartbeat every 10 seconds (it also
# writes the kitchen:heartbeat key in Redis). Healthy = the file changed in the last minute.
HEALTHCHECK --interval=20s --timeout=5s --start-period=40s --retries=3 \
    CMD test -n "$(find /tmp/kitchen-heartbeat -mmin -1 2>/dev/null)" || exit 1

# MaxRAMPercentage makes the JVM size its heap from the container memory limit.
ENTRYPOINT ["java", "-XX:MaxRAMPercentage=75", "-jar", "app.jar"]
