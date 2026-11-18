"""pytest tests for menu-api."""
import json
from unittest.mock import MagicMock, patch

from fastapi.testclient import TestClient

from main import app

client = TestClient(app)


def _rows():
    return [
        {"id": 1, "name": "Chicken Biryani", "description": "Spiced rice", "price": 450.0, "category": "Main Course"},
        {"id": 2, "name": "Naan", "description": "Flatbread", "price": 50.0, "category": "Bread"},
    ]


@patch("main.get_redis")
@patch("main.get_db")
def test_menu_cache_miss(mock_db, mock_redis):
    r = MagicMock()
    r.get.return_value = None
    mock_redis.return_value = r

    cur = MagicMock()
    cur.fetchall.return_value = _rows()
    conn = MagicMock()
    conn.cursor.return_value = cur
    mock_db.return_value = conn

    response = client.get("/menu")

    assert response.status_code == 200
    assert response.json()[0]["name"] == "Chicken Biryani"
    r.setex.assert_called_once()


@patch("main.get_db")
@patch("main.get_redis")
def test_menu_cache_hit(mock_redis, mock_db):
    r = MagicMock()
    r.get.return_value = json.dumps(_rows())
    mock_redis.return_value = r

    response = client.get("/menu")

    assert response.status_code == 200
    assert len(response.json()) == 2
    mock_db.assert_not_called()


@patch("main.get_redis")
@patch("main.get_db")
def test_health_ok(mock_db, mock_redis):
    mock_db.return_value = MagicMock()
    r = MagicMock()
    r.ping.return_value = True
    mock_redis.return_value = r

    response = client.get("/health")

    assert response.status_code == 200
    assert response.json()["status"] == "ok"


@patch("main.get_redis")
@patch("main.get_db")
def test_health_fails_when_redis_down(mock_db, mock_redis):
    mock_db.return_value = MagicMock()
    r = MagicMock()
    r.ping.side_effect = ConnectionError("down")
    mock_redis.return_value = r

    assert client.get("/health").status_code == 503
