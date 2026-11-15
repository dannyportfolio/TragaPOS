package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"pos-engine/internal/config"
	deliveryHttp "pos-engine/internal/delivery/http"
	"pos-engine/internal/delivery/ws"
)

func TestSetupRouter_AuthAndMetrics(t *testing.T) {
	cfg := &config.Config{
		AppMode:       config.ModeEdge,
		APIAuthToken:  "secret-api-token",
		EdgeAuthToken: "secret-edge-token",
	}
	hub := ws.NewHub()
	handler := &deliveryHttp.Handler{}
	router := deliveryHttp.SetupRouter(handler, hub, cfg)

	// /metrics should be accessible without auth
	reqMetrics := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	recMetrics := httptest.NewRecorder()
	router.ServeHTTP(recMetrics, reqMetrics)
	if recMetrics.Code != http.StatusOK {
		t.Fatalf("expected 200 on /metrics, got %d", recMetrics.Code)
	}

	// /v1/catalog/items without token should return 401
	reqUnauth := httptest.NewRequest(http.MethodGet, "/v1/catalog/items", nil)
	recUnauth := httptest.NewRecorder()
	router.ServeHTTP(recUnauth, reqUnauth)
	if recUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on /v1/catalog/items without token, got %d", recUnauth.Code)
	}

	// /v1/sync/pull with API token only (missing Edge token) should return 401
	reqSyncUnauth := httptest.NewRequest(http.MethodGet, "/v1/sync/pull", nil)
	reqSyncUnauth.Header.Set("Authorization", "Bearer secret-api-token")
	recSyncUnauth := httptest.NewRecorder()
	router.ServeHTTP(recSyncUnauth, reqSyncUnauth)
	if recSyncUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on /v1/sync/pull without edge token, got %d", recSyncUnauth.Code)
	}
}
