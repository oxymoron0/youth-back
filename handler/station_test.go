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

type mockStationRepo struct {
	stations    []model.Station
	total       int
	detail      *model.StationDetail
	searchRes   []model.SearchResult
	nearbyRes   json.RawMessage
	geojsonRes  json.RawMessage
	geojsonTotal int
	err         error
}

func (m *mockStationRepo) List(_ context.Context, _, _ int) ([]model.Station, int, error) {
	return m.stations, m.total, m.err
}

func (m *mockStationRepo) ListGeoJSON(_ context.Context, _, _ int) (json.RawMessage, int, error) {
	return m.geojsonRes, m.geojsonTotal, m.err
}

func (m *mockStationRepo) ListGeoJSONAll(_ context.Context, _ *int) (json.RawMessage, error) {
	return m.geojsonRes, m.err
}

func (m *mockStationRepo) GetByID(_ context.Context, _ int) (*model.StationDetail, error) {
	return m.detail, m.err
}

func (m *mockStationRepo) Search(_ context.Context, _ string, _ int) ([]model.SearchResult, error) {
	return m.searchRes, m.err
}

func (m *mockStationRepo) Nearby(_ context.Context, _, _ float64, _, _ int) (json.RawMessage, error) {
	return m.nearbyRes, m.err
}

func setupStationRouter(mock *mockStationRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewStationHandler(mock)
	v1 := r.Group("/api/v1")
	v1.GET("/stations", h.List)
	v1.GET("/stations/:id", h.GetByID)
	v1.GET("/stations/search", h.Search)
	v1.GET("/stations/nearby", h.Nearby)
	return r
}

func TestStationList_JSON(t *testing.T) {
	mock := &mockStationRepo{
		stations: []model.Station{
			{StationID: 1, StationCode: "S001", StationName: "서울역"},
			{StationID: 2, StationCode: "S002", StationName: "시청"},
		},
		total: 2,
	}
	r := setupStationRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/stations?per_page=2", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("X-Total-Count") != "2" {
		t.Fatalf("expected X-Total-Count=2, got %s", w.Header().Get("X-Total-Count"))
	}

	var stations []model.Station
	if err := json.Unmarshal(w.Body.Bytes(), &stations); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if len(stations) != 2 {
		t.Fatalf("expected 2 stations, got %d", len(stations))
	}
}

func TestStationList_GeoJSON(t *testing.T) {
	fc := json.RawMessage(`{"type":"FeatureCollection","features":[]}`)
	mock := &mockStationRepo{
		geojsonRes:   fc,
		geojsonTotal: 10,
	}
	r := setupStationRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/stations?format=geojson", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/geo+json" {
		t.Fatalf("expected content-type geo+json, got %s", ct)
	}
}

func TestStationGetByID_Found(t *testing.T) {
	mock := &mockStationRepo{
		detail: &model.StationDetail{
			StationID:   1,
			StationCode: "S001",
			StationName: "서울역",
			Lines:       []model.StationLine{},
			Transfers:   []model.StationTransfer{},
			Exits:       []model.StationExit{},
		},
	}
	r := setupStationRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/stations/1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestStationGetByID_NotFound(t *testing.T) {
	mock := &mockStationRepo{detail: nil}
	r := setupStationRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/stations/999", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestStationGetByID_InvalidID(t *testing.T) {
	mock := &mockStationRepo{}
	r := setupStationRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/stations/abc", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestStationSearch_OK(t *testing.T) {
	mock := &mockStationRepo{
		searchRes: []model.SearchResult{
			{StationID: 1, StationName: "강남", Lines: json.RawMessage(`[]`)},
		},
	}
	r := setupStationRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/stations/search?q=강남", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestStationSearch_EmptyQuery(t *testing.T) {
	mock := &mockStationRepo{}
	r := setupStationRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/stations/search?q=", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestStationNearby_OK(t *testing.T) {
	fc := json.RawMessage(`{"type":"FeatureCollection","features":[]}`)
	mock := &mockStationRepo{nearbyRes: fc}
	r := setupStationRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/stations/nearby?lon=126.97&lat=37.55&radius=1000", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestStationNearby_MissingParams(t *testing.T) {
	mock := &mockStationRepo{}
	r := setupStationRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/stations/nearby", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestStationNearby_InvalidLon(t *testing.T) {
	mock := &mockStationRepo{}
	r := setupStationRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/stations/nearby?lon=abc&lat=37.55", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
