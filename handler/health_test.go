package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type mockHealthChecker struct {
	err error
}

func (m *mockHealthChecker) Ping(_ context.Context) error {
	return m.err
}

func TestHealth_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/healthz", Health(&mockHealthChecker{}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if body := w.Body.String(); body == "" {
		t.Fatal("empty body")
	}
}

func TestHealth_Unhealthy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/healthz", Health(&mockHealthChecker{err: errors.New("connection refused")}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}
