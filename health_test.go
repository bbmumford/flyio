package flyio

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler_OK(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	HealthHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("health handler status = %d, want 200", rec.Code)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("health response body must not be empty (carries build info + uptime)")
	}
}
