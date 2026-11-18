# ---- Build stage ----
FROM python:3.12-alpine AS builder

WORKDIR /build
COPY requirements.txt .
RUN pip install --no-cache-dir --prefix=/install -r requirements.txt

# ---- Final stage ----
FROM python:3.12-alpine

ARG APP_VERSION=dev
ENV APP_VERSION=${APP_VERSION} \
    PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1
LABEL org.opencontainers.image.revision="${APP_VERSION}"

RUN apk upgrade --no-cache && \
    addgroup --system app && adduser --system --ingroup app app

WORKDIR /app
COPY --from=builder /install /usr/local
COPY --chown=app:app main.py .

USER app
EXPOSE 8000

# /health reports the version too: {"status":"ok","version":"<commit>"}
HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
    CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8000/health')"

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000", "--workers", "2"]
