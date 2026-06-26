package sync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGeocode_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("query"); got == "" {
			t.Errorf("missing query param")
		}
		if r.Header.Get("x-ncp-apigw-api-key-id") != "id" || r.Header.Get("x-ncp-apigw-api-key") != "secret" {
			t.Errorf("missing/incorrect auth headers")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"OK","addresses":[{"x":"127.0509544","y":"37.5806024"}]}`))
	}))
	defer ts.Close()

	g := NewGeocoder("id", "secret").WithHTTPClient(ts.Client()).WithURL(ts.URL)
	lng, lat, ok, err := g.Geocode(context.Background(), "서울 동대문구 전농동 127-359")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true")
	}
	if lng != 127.0509544 || lat != 37.5806024 {
		t.Errorf("unexpected coords: lng=%v lat=%v", lng, lat)
	}
}

func TestGeocode_NoMatch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"status":"OK","addresses":[]}`))
	}))
	defer ts.Close()

	g := NewGeocoder("id", "secret").WithHTTPClient(ts.Client()).WithURL(ts.URL)
	_, _, ok, err := g.Geocode(context.Background(), "존재하지 않는 주소")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected ok=false on no match")
	}
}

func TestGeocode_ErrorStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"status":"INVALID_REQUEST","errorMessage":"bad"}`))
	}))
	defer ts.Close()

	g := NewGeocoder("id", "secret").WithHTTPClient(ts.Client()).WithURL(ts.URL)
	if _, _, _, err := g.Geocode(context.Background(), "x"); err == nil {
		t.Fatal("expected error on non-OK status")
	}
}

func TestGeocoder_EnabledNilSafe(t *testing.T) {
	var g *Geocoder
	if g.Enabled() {
		t.Error("nil geocoder must not be enabled")
	}
	if NewGeocoder("", "").Enabled() {
		t.Error("empty credentials must not be enabled")
	}
	if !NewGeocoder("id", "secret").Enabled() {
		t.Error("configured geocoder should be enabled")
	}
}
