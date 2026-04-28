package sync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kr-metro-api/model"
)

type mockHousingRepo struct {
	upsertUpdated int
	upsertNew     int
	upsertErr     error
	upsertCalled  bool
	saveCalled    bool
	saveResult    model.HousingSyncResult
	saveErr       error
}

func (m *mockHousingRepo) List(_ context.Context) ([]model.HousingListItem, error) {
	return nil, nil
}

func (m *mockHousingRepo) GetByHomeCode(_ context.Context, _ string) (*model.HousingDetail, error) {
	return nil, nil
}

func (m *mockHousingRepo) NearbyStations(_ context.Context, _ string, _ int) ([]model.NearbyStation, error) {
	return nil, nil
}

func (m *mockHousingRepo) UpsertFromListAPI(_ context.Context, _ []model.HousingSyncItem) (int, int, error) {
	m.upsertCalled = true
	return m.upsertUpdated, m.upsertNew, m.upsertErr
}

func (m *mockHousingRepo) SaveSyncResult(_ context.Context, result model.HousingSyncResult) error {
	m.saveCalled = true
	m.saveResult = result
	return m.saveErr
}

func (m *mockHousingRepo) LatestSyncResult(_ context.Context) (*model.HousingSyncResult, error) {
	return nil, nil
}

func (m *mockHousingRepo) RecentSyncHistory(_ context.Context, _ int) ([]model.HousingSyncResult, error) {
	return nil, nil
}

func newTestServer(response string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(response))
	}))
}

func TestRunOnce_Success(t *testing.T) {
	ts := newTestServer(`{
		"resultList": [
			{"homeCode": "H001", "homeName": "A", "supplyStatus": "02"},
			{"homeCode": "H002", "homeName": "B", "supplyStatus": "03"}
		]
	}`)
	defer ts.Close()

	repo := &mockHousingRepo{upsertUpdated: 1, upsertNew: 1}
	client := NewHousingClient().WithHTTPClient(ts.Client()).WithListURL(ts.URL)
	syncer := NewHousingSync(client, repo)

	result := syncer.RunOnce(context.Background())

	if result.Error != "" {
		t.Fatalf("unexpected error: %s", result.Error)
	}
	if result.FetchedCount != 2 {
		t.Errorf("expected fetched 2, got %d", result.FetchedCount)
	}
	if result.UpdatedCount != 1 {
		t.Errorf("expected updated 1, got %d", result.UpdatedCount)
	}
	if result.NewCount != 1 {
		t.Errorf("expected new 1, got %d", result.NewCount)
	}
	if !repo.upsertCalled {
		t.Error("UpsertFromListAPI was not called")
	}
	if !repo.saveCalled {
		t.Error("SaveSyncResult was not called")
	}
	if repo.saveResult.FetchedCount != 2 || repo.saveResult.UpdatedCount != 1 || repo.saveResult.NewCount != 1 {
		t.Errorf("saved result mismatch: %+v", repo.saveResult)
	}
	if repo.saveResult.DurationMs < 0 {
		t.Errorf("expected non-negative duration, got %d", repo.saveResult.DurationMs)
	}
}

func TestRunOnce_PersistsOnFetchError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	repo := &mockHousingRepo{}
	client := NewHousingClient().WithHTTPClient(ts.Client()).WithListURL(ts.URL)
	syncer := NewHousingSync(client, repo)

	syncer.RunOnce(context.Background())

	if !repo.saveCalled {
		t.Error("SaveSyncResult should be called even on fetch error")
	}
	if repo.saveResult.Error == "" {
		t.Error("saved result should contain error message")
	}
}

func TestRunOnce_FetchError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	repo := &mockHousingRepo{}
	client := NewHousingClient().WithHTTPClient(ts.Client()).WithListURL(ts.URL)
	syncer := NewHousingSync(client, repo)

	result := syncer.RunOnce(context.Background())

	if result.Error == "" {
		t.Fatal("expected error")
	}
	if repo.upsertCalled {
		t.Error("UpsertFromListAPI should not be called on fetch error")
	}
}

func TestRunOnce_UpsertError(t *testing.T) {
	ts := newTestServer(`{
		"resultList": [{"homeCode": "H001", "homeName": "A", "supplyStatus": "02"}]
	}`)
	defer ts.Close()

	repo := &mockHousingRepo{upsertErr: context.DeadlineExceeded}
	client := NewHousingClient().WithHTTPClient(ts.Client()).WithListURL(ts.URL)
	syncer := NewHousingSync(client, repo)

	result := syncer.RunOnce(context.Background())

	if result.Error == "" {
		t.Fatal("expected error")
	}
	if result.FetchedCount != 1 {
		t.Errorf("expected fetched 1, got %d", result.FetchedCount)
	}
}

func TestLastResult(t *testing.T) {
	ts := newTestServer(`{
		"resultList": [{"homeCode": "H001", "homeName": "A", "supplyStatus": "02"}]
	}`)
	defer ts.Close()

	repo := &mockHousingRepo{upsertUpdated: 1}
	client := NewHousingClient().WithHTTPClient(ts.Client()).WithListURL(ts.URL)
	syncer := NewHousingSync(client, repo)

	if syncer.LastResult() != nil {
		t.Fatal("expected nil before first run")
	}

	syncer.RunOnce(context.Background())

	last := syncer.LastResult()
	if last == nil {
		t.Fatal("expected non-nil after run")
	}
	if last.FetchedCount != 1 {
		t.Errorf("expected fetched 1, got %d", last.FetchedCount)
	}
}

func TestStartStopsOnCancel(t *testing.T) {
	ts := newTestServer(`{
		"resultList": [{"homeCode": "H001", "homeName": "A", "supplyStatus": "02"}]
	}`)
	defer ts.Close()

	repo := &mockHousingRepo{}
	client := NewHousingClient().WithHTTPClient(ts.Client()).WithListURL(ts.URL)
	syncer := NewHousingSync(client, repo, WithInterval(100*time.Millisecond))

	ctx, cancel := context.WithTimeout(context.Background(), 350*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		syncer.Start(ctx)
		close(done)
	}()

	select {
	case <-done:
		// OK: Start returned after context cancellation
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}

	if !repo.upsertCalled {
		t.Error("expected at least one sync run")
	}
}
