package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"kr-metro-api/model"
)

type mockTransferRepo struct {
	transfers []model.Transfer
	err       error
}

func (m *mockTransferRepo) GetByStation(_ context.Context, _ int) ([]model.Transfer, error) {
	return m.transfers, m.err
}

func setupTransferRouter(mock *mockTransferRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewTransferHandler(mock)
	v1 := r.Group("/api/v1")
	v1.GET("/transfers/:station_id", h.GetByStation)
	return r
}

func TestTransferGetByStation_OK(t *testing.T) {
	mock := &mockTransferRepo{
		transfers: []model.Transfer{
			{TransferID: 1, FromStationID: 1, FromStation: "서울역",
				FromLineID: 1, FromLineName: "1호선",
				ToStationID: 1, ToStation: "서울역",
				ToLineID: 2, ToLineName: "4호선"},
		},
	}
	r := setupTransferRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/transfers/1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var transfers []model.Transfer
	if err := json.Unmarshal(w.Body.Bytes(), &transfers); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(transfers) != 1 {
		t.Fatalf("expected 1 transfer, got %d", len(transfers))
	}
}

func TestTransferGetByStation_Empty(t *testing.T) {
	mock := &mockTransferRepo{transfers: []model.Transfer{}}
	r := setupTransferRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/transfers/999", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestTransferGetByStation_InvalidID(t *testing.T) {
	mock := &mockTransferRepo{}
	r := setupTransferRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/transfers/abc", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestTransferGetByStation_Error(t *testing.T) {
	mock := &mockTransferRepo{err: errors.New("db error")}
	r := setupTransferRouter(mock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/transfers/1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
