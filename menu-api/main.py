"""menu-api: FastAPI service.

Reads menu items from Postgres and caches them in Redis.
"""
import json
import os

import psycopg2
import psycopg2.extras
import redis as redis_lib
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware

app = FastAPI(title="Dhaba Menu API", version=os.getenv("APP_VERSION", "dev"))

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["GET", "DELETE"],
    allow_headers=["*"],
)

DATABASE_URL = os.getenv("DATABASE_URL", "postgres://dhaba:dhaba@postgres:5432/dhaba")
REDIS_URL = os.getenv("REDIS_URL", "redis://redis:6379")
CACHE_TTL = int(os.getenv("CACHE_TTL_SECONDS", "300"))
CACHE_KEY = "menu:all"


def get_db():
    return psycopg2.connect(DATABASE_URL, cursor_factory=psycopg2.extras.RealDictCursor)


def get_redis():
    return redis_lib.from_url(REDIS_URL, decode_responses=True)


@app.get("/health")
def health():
    """Check that Postgres and Redis are reachable."""
    try:
        get_db().close()
    except Exception as exc:
        raise HTTPException(status_code=503, detail=f"postgres unavailable: {exc}") from exc
    try:
        get_redis().ping()
    except Exception as exc:
        raise HTTPException(status_code=503, detail=f"redis unavailable: {exc}") from exc
    return {"status": "ok", "version": app.version}


@app.get("/menu")
def get_menu():
    """Return available menu items, from Redis when the cache is warm."""
    r = get_redis()
    cached = r.get(CACHE_KEY)
    if cached:
        return json.loads(cached)

    conn = get_db()
    try:
        cur = conn.cursor()
        cur.execute(
            """
            SELECT id, name, description, price, category
            FROM   menu_items
            WHERE  available = true
            ORDER  BY category, name
            """
        )
        items = [dict(row) for row in cur.fetchall()]
        for item in items:
            item["price"] = float(item["price"])  # Decimal is not JSON serialisable
    finally:
        conn.close()

    r.setex(CACHE_KEY, CACHE_TTL, json.dumps(items))
    return items


@app.delete("/menu/cache")
def invalidate_cache():
    """Clear the menu cache after a menu change."""
    get_redis().delete(CACHE_KEY)
    return {"message": "cache cleared"}
