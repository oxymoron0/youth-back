package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"kr-metro-api/model"
	"kr-metro-api/repository"
)

// Ensure mockHousingRepo implements the full interface.
var _ repository.HousingRepository = (*mockHousingRepo)(nil)

type mockHousingRepo struct {
	listItems      []model.HousingListItem
	detail         *model.HousingDetail
	nearbyStations []model.NearbyStation
	latestSync     *model.HousingSyncResult
	historySync    []model.HousingSyncResult
	image          *model.HousingImage
	err            error
}

func (m *mockHousingRepo) List(_ context.Context) ([]model.HousingListItem, error) {
	return m.listItems, m.err
}

func (m *mockHousingRepo) GetByHomeCode(_ context.Context, _ string) (*model.HousingDetail, error) {
	return m.detail, m.err
}

func (m *mockHousingRepo) NearbyStations(_ context.Context, _ string, _ int) ([]model.NearbyStation, error) {
	return m.nearbyStations, m.err
}

func (m *mockHousingRepo) UpsertFromListAPI(_ context.Context, _ []model.HousingSyncItem) (int, []string, error) {
	return 0, nil, nil
}

func (m *mockHousingRepo) HousingsMissingCoords(_ context.Context) ([]model.HousingCoordTarget, error) {
	return nil, nil
}

func (m *mockHousingRepo) UpdateHousingDetail(_ context.Context, _ string, _ model.HousingDetailFields) error {
	return nil
}

func (m *mockHousingRepo) SaveSyncResult(_ context.Context, _ model.HousingSyncResult) error {
	return nil
}

func (m *mockHousingRepo) LatestSyncResult(_ context.Context) (*model.HousingSyncResult, error) {
	return m.latestSync, m.err
}

func (m *mockHousingRepo) RecentSyncHistory(_ context.Context, _ int) ([]model.HousingSyncResult, error) {
	return m.historySync, m.err
}

func (m *mockHousingRepo) ImageRef(_ context.Context, _ string) (string, int, bool, error) {
	return "", 0, false, nil
}

func (m *mockHousingRepo) UpsertImage(_ context.Context, _ model.HousingImage) error {
	return nil
}

func (m *mockHousingRepo) GetImage(_ context.Context, _ string) (*model.HousingImage, error) {
	return m.image, m.err
}

func setupHousingRouter(mock *mockHousingRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHousingHandler(mock)
	v1 := r.Group("/api/v1")
	v1.GET("/housings", h.List)
	v1.GET("/housings/:home_code", h.GetByHomeCode)
	v1.GET("/housings/:home_code/image", h.GetImage)
	v1.GET("/housings/:home_code/nearby-stations", h.NearbyStations)
	return r
}

func TestHousingList_OK(t *testing.T) {
	mock := &mockHousingRepo{
		listItems: []model.HousingListItem{
			{HomeCode: "H001", HomeName: "행복주택A", SupplyStatus: "공급중"},
			{HomeCode: "H002", HomeName: "행복주택B", SupplyStatus: "대기"},
		},
	}
	r := setupHousingRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/housings", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var items []model.HousingListItem
	if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestHousingList_Empty(t *testing.T) {
	mock := &mockHousingRepo{
		listItems: []model.HousingListItem{},
	}
	r := setupHousingRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/housings", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var items []model.HousingListItem
	if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
	// Verify it's an empty array, not null
	if string(w.Body.Bytes()) == "null" {
		t.Fatal("expected empty array [], got null")
	}
}

func TestHousingGetByHomeCode_Found(t *testing.T) {
	mock := &mockHousingRepo{
		detail: &model.HousingDetail{
			HousingID:    1,
			HomeCode:     "H001",
			HomeName:     "행복주택A",
			SupplyStatus: "공급중",
		},
	}
	r := setupHousingRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/housings/H001", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var detail model.HousingDetail
	if err := json.Unmarshal(w.Body.Bytes(), &detail); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if detail.HomeCode != "H001" {
		t.Fatalf("expected home_code H001, got %s", detail.HomeCode)
	}
}

func TestHousingGetByHomeCode_NotFound(t *testing.T) {
	mock := &mockHousingRepo{detail: nil}
	r := setupHousingRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/housings/NONEXIST", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHousingGetImage_OK(t *testing.T) {
	mock := &mockHousingRepo{
		image: &model.HousingImage{
			HomeCode:    "H001",
			ContentType: "image/png",
			ETag:        "deadbeef",
			Data:        []byte{0x89, 0x50, 0x4E, 0x47},
		},
	}
	r := setupHousingRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/housings/H001/image", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("expected image/png, got %s", ct)
	}
	if etag := w.Header().Get("ETag"); etag != `"deadbeef"` {
		t.Errorf("expected quoted etag, got %s", etag)
	}
	if cc := w.Header().Get("Cache-Control"); cc == "" {
		t.Error("expected Cache-Control header")
	}
	if w.Body.Len() != 4 {
		t.Errorf("expected 4 bytes, got %d", w.Body.Len())
	}
}

func TestHousingGetImage_SniffsNonImageContentType(t *testing.T) {
	pngMagic := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	mock := &mockHousingRepo{
		image: &model.HousingImage{
			HomeCode:    "H001",
			ContentType: "application/octet-stream", // source mislabel
			ETag:        "abc",
			Data:        pngMagic,
		},
	}
	r := setupHousingRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/housings/H001/image", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("expected sniffed image/png, got %s", ct)
	}
}

func TestHousingGetImage_NotModified(t *testing.T) {
	mock := &mockHousingRepo{
		image: &model.HousingImage{
			HomeCode: "H001", ContentType: "image/png", ETag: "deadbeef", Data: []byte{0x89},
		},
	}
	r := setupHousingRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/housings/H001/image", nil)
	req.Header.Set("If-None-Match", `"deadbeef"`)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotModified {
		t.Fatalf("expected 304, got %d", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Errorf("expected empty body on 304, got %d bytes", w.Body.Len())
	}
}

func TestHousingGetImage_NotFound(t *testing.T) {
	mock := &mockHousingRepo{image: nil}
	r := setupHousingRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/housings/NONEXIST/image", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHousingNearbyStations_OK(t *testing.T) {
	lat := 37.55
	lon := 126.97
	mock := &mockHousingRepo{
		detail: &model.HousingDetail{
			HousingID:    1,
			HomeCode:     "H001",
			HomeName:     "행복주택A",
			SupplyStatus: "공급중",
		},
		nearbyStations: []model.NearbyStation{
			{StationID: 1, StationName: "서울역", LineNames: "1호선, 4호선", DistanceM: 120, Latitude: &lat, Longitude: &lon},
		},
	}
	r := setupHousingRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/housings/H001/nearby-stations?distance=500", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var stations []model.NearbyStation
	if err := json.Unmarshal(w.Body.Bytes(), &stations); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if len(stations) != 1 {
		t.Fatalf("expected 1 station, got %d", len(stations))
	}
}

func TestHousingNearbyStations_NotFound(t *testing.T) {
	mock := &mockHousingRepo{detail: nil}
	r := setupHousingRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/housings/NONEXIST/nearby-stations", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHousingNearbyStations_DefaultDistance(t *testing.T) {
	mock := &mockHousingRepo{
		detail: &model.HousingDetail{
			HousingID:    1,
			HomeCode:     "H001",
			HomeName:     "행복주택A",
			SupplyStatus: "공급중",
		},
		nearbyStations: []model.NearbyStation{},
	}
	r := setupHousingRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/housings/H001/nearby-stations", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var stations []model.NearbyStation
	if err := json.Unmarshal(w.Body.Bytes(), &stations); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if len(stations) != 0 {
		t.Fatalf("expected 0 stations, got %d", len(stations))
	}
}

func TestHousingNearbyStations_CustomDistance(t *testing.T) {
	lat := 37.55
	lon := 126.97
	mock := &mockHousingRepo{
		detail: &model.HousingDetail{
			HousingID:    1,
			HomeCode:     "H001",
			HomeName:     "행복주택A",
			SupplyStatus: "공급중",
		},
		nearbyStations: []model.NearbyStation{
			{StationID: 1, StationName: "서울역", LineNames: "1호선", DistanceM: 300, Latitude: &lat, Longitude: &lon},
			{StationID: 2, StationName: "시청", LineNames: "1호선, 2호선", DistanceM: 800, Latitude: &lat, Longitude: &lon},
		},
	}
	r := setupHousingRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/housings/H001/nearby-stations?distance=1000", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var stations []model.NearbyStation
	if err := json.Unmarshal(w.Body.Bytes(), &stations); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if len(stations) != 2 {
		t.Fatalf("expected 2 stations, got %d", len(stations))
	}
}
