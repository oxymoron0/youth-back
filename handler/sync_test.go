package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"kr-metro-api/model"
)

func setupSyncRouter(mock *mockHousingRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewSyncHandler(nil, mock, "")
	v1 := r.Group("/api/v1")
	v1.GET("/housings/sync/latest", h.PublicLatestStatus)
	v1.GET("/housings/sync/history", h.PublicHistory)
	return r
}

func TestPublicLatestStatus_FromDB(t *testing.T) {
	now := time.Now()
	mock := &mockHousingRepo{
		latestSync: &model.HousingSyncResult{
			FetchedCount: 81,
			UpdatedCount: 80,
			NewCount:     1,
			Duration:     "400ms",
			DurationMs:   400,
			StartedAt:    now,
			CompletedAt:  now.Add(400 * time.Millisecond),
		},
	}
	r := setupSyncRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/housings/sync/latest", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Status string                   `json:"status"`
		Result model.HousingSyncResult  `json:"result"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if resp.Result.FetchedCount != 81 {
		t.Errorf("expected fetched 81, got %d", resp.Result.FetchedCount)
	}
}

func TestPublicLatestStatus_NoHistory(t *testing.T) {
	mock := &mockHousingRepo{latestSync: nil}
	r := setupSyncRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/housings/sync/latest", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if resp["message"] == nil {
		t.Error("expected message field for empty history")
	}
}

func TestPublicHistory_OK(t *testing.T) {
	now := time.Now()
	mock := &mockHousingRepo{
		historySync: []model.HousingSyncResult{
			{FetchedCount: 81, UpdatedCount: 81, StartedAt: now},
			{FetchedCount: 81, UpdatedCount: 81, StartedAt: now.Add(-30 * time.Minute)},
		},
	}
	r := setupSyncRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/housings/sync/history?limit=2", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Status  string                    `json:"status"`
		Count   int                       `json:"count"`
		Results []model.HousingSyncResult `json:"results"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if resp.Count != 2 {
		t.Errorf("expected count 2, got %d", resp.Count)
	}
	if len(resp.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(resp.Results))
	}
}

func TestPublicHistory_EmptyResults(t *testing.T) {
	mock := &mockHousingRepo{historySync: nil}
	r := setupSyncRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/housings/sync/history", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Status  string                    `json:"status"`
		Count   int                       `json:"count"`
		Results []model.HousingSyncResult `json:"results"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if resp.Count != 0 {
		t.Errorf("expected count 0, got %d", resp.Count)
	}
}
