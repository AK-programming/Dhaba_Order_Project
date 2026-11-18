package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInfoHandler(t *testing.T) {
	rr := httptest.NewRecorder()
	infoHandler(rr, httptest.NewRequest(http.MethodGet, "/info", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "version") || !strings.Contains(body, "pod") {
		t.Errorf("response missing version or pod: %s", body)
	}
}

func postOrder(body string) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	createOrderHandler(rr, httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(body)))
	return rr
}

func TestCreateOrderBadJSON(t *testing.T) {
	if rr := postOrder("not-json"); rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCreateOrderMissingName(t *testing.T) {
	body := `{"items":[{"menu_item_id":1,"quantity":1}]}`
	if rr := postOrder(body); rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCreateOrderEmptyItems(t *testing.T) {
	if rr := postOrder(`{"customer_name":"Ali","items":[]}`); rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCreateOrderBadQuantity(t *testing.T) {
	body := `{"customer_name":"Ali","items":[{"menu_item_id":1,"quantity":0}]}`
	if rr := postOrder(body); rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
