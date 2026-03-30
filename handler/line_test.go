package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"kr-metro-api/model"
)

type mockLineRepo struct {
	lines      []model.Line
	stationsFC json.RawMessage
	geometry   json.RawMessage
	err        error
}

func (m *mockLineRepo) List(_ context.Context) ([]model.Line, error) {
	return m.lines, m.err
}

func (m *mockLineRepo) ListStations(_ context.Context, _ int) (json.RawMessage, error) {
	return m.stationsFC, m.err
}

func (m *mockLineRepo) GetGeometry(_ context.Context, _ int) (json.RawMessage, error) {
	return m.geometry, m.err
}

func setupLineRouter(mock *mockLineRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewLineHandler(mock)
	v1 := r.Group("/api/v1")
	v1.GET("/lines", h.List)
	v1.GET("/lines/:id/stations", h.ListStations)
	v1.GET("/lines/:id/geometry", h.GetGeometry)
	return r
}

func TestLineList_OK(t *testing.T) {
	mock := &mockLineRepo{
		lines: []model.Line{
			{LineID: 1, LineCode: "S1101", LineName: "1호선", StationCount: 10},
		},
	}
	r := setupLineRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/lines", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var lines []model.Line
	if err := json.Unmarshal(w.Body.Bytes(), &lines); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
}

func TestLineListStations_OK(t *testing.T) {
	fc := json.RawMessage(`{"type":"FeatureCollection","features":[]}`)
	mock := &mockLineRepo{stationsFC: fc}
	r := setupLineRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/lines/1/stations", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/geo+json" {
		t.Fatalf("expected content-type geo+json, got %s", ct)
	}
}

func TestLineListStations_InvalidID(t *testing.T) {
	mock := &mockLineRepo{}
	r := setupLineRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/lines/abc/stations", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestLineListStations_NotFound(t *testing.T) {
	mock := &mockLineRepo{stationsFC: nil}
	r := setupLineRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/lines/999/stations", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestLineGeometry_OK(t *testing.T) {
	geom := json.RawMessage(`{"type":"Feature","geometry":{"type":"LineString","coordinates":[[126.97,37.55]]}}`)
	mock := &mockLineRepo{geometry: geom}
	r := setupLineRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/lines/1/geometry", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestLineGeometry_NotFound(t *testing.T) {
	mock := &mockLineRepo{geometry: nil}
	r := setupLineRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/lines/999/geometry", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
