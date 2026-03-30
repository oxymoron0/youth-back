package sync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchList_OK(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("User-Agent") != userAgent {
			t.Errorf("expected User-Agent %s, got %s", userAgent, r.Header.Get("User-Agent"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"resultList": [
				{"homeCode": "H001", "homeName": "테스트A", "supplyStatus": "02"},
				{"homeCode": "H002", "homeName": "테스트B", "supplyStatus": "03"}
			]
		}`))
	}))
	defer ts.Close()

	client := NewHousingClient().WithHTTPClient(ts.Client()).WithListURL(ts.URL)
	items, err := client.FetchList(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].HomeCode != "H001" {
		t.Errorf("expected HomeCode H001, got %s", items[0].HomeCode)
	}
}

func TestFetchList_EmptyResult(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"resultList": []}`))
	}))
	defer ts.Close()

	client := NewHousingClient().WithHTTPClient(ts.Client()).WithListURL(ts.URL)
	_, err := client.FetchList(context.Background())
	if err == nil {
		t.Fatal("expected error for empty result list")
	}
}

func TestFetchList_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := NewHousingClient().WithHTTPClient(ts.Client()).WithListURL(ts.URL)
	_, err := client.FetchList(context.Background())
	if err == nil {
		t.Fatal("expected error for server error")
	}
}

func TestFetchList_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`not json`))
	}))
	defer ts.Close()

	client := NewHousingClient().WithHTTPClient(ts.Client()).WithListURL(ts.URL)
	_, err := client.FetchList(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestFetchList_MoneyFields(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// API returns mixed types for money fields (int or string)
		w.Write([]byte(`{
			"resultList": [
				{"homeCode": "H001", "homeName": "A", "supplyStatus": "02", "moneyDepositLow": 5000, "moneyRentalLow": "200"}
			]
		}`))
	}))
	defer ts.Close()

	client := NewHousingClient().WithHTTPClient(ts.Client()).WithListURL(ts.URL)
	items, err := client.FetchList(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}
