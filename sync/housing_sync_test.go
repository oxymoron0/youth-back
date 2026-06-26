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
	upsertUpdated  int
	upsertNewCodes []string
	upsertErr      error
	upsertCalled   bool
	saveCalled     bool
	saveResult     model.HousingSyncResult
	saveErr        error
	latest         *model.HousingSyncResult
	// image tracking
	imageRefs map[string]struct {
		fileID string
		fileSn int
	}
	upsertedImages []model.HousingImage
	detailUpdates  []string
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

func (m *mockHousingRepo) UpsertFromListAPI(_ context.Context, _ []model.HousingSyncItem) (int, []string, error) {
	m.upsertCalled = true
	return m.upsertUpdated, m.upsertNewCodes, m.upsertErr
}

func (m *mockHousingRepo) HousingsMissingCoords(_ context.Context) ([]model.HousingCoordTarget, error) {
	return nil, nil
}

func (m *mockHousingRepo) UpdateHousingDetail(_ context.Context, homeCode string, _ model.HousingDetailFields) error {
	m.detailUpdates = append(m.detailUpdates, homeCode)
	return nil
}

func (m *mockHousingRepo) SaveSyncResult(_ context.Context, result model.HousingSyncResult) error {
	m.saveCalled = true
	m.saveResult = result
	return m.saveErr
}

func (m *mockHousingRepo) LatestSyncResult(_ context.Context) (*model.HousingSyncResult, error) {
	return m.latest, nil
}

func (m *mockHousingRepo) RecentSyncHistory(_ context.Context, _ int) ([]model.HousingSyncResult, error) {
	return nil, nil
}

func (m *mockHousingRepo) ImageRef(_ context.Context, homeCode string) (string, int, bool, error) {
	if m.imageRefs == nil {
		return "", 0, false, nil
	}
	ref, ok := m.imageRefs[homeCode]
	if !ok {
		return "", 0, false, nil
	}
	return ref.fileID, ref.fileSn, true, nil
}

func (m *mockHousingRepo) UpsertImage(_ context.Context, img model.HousingImage) error {
	m.upsertedImages = append(m.upsertedImages, img)
	return nil
}

func (m *mockHousingRepo) GetImage(_ context.Context, _ string) (*model.HousingImage, error) {
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

	repo := &mockHousingRepo{upsertUpdated: 1, upsertNewCodes: []string{"H002"}}
	client := NewHousingClient().WithHTTPClient(ts.Client()).WithListURL(ts.URL).WithDetailURL(ts.URL)
	syncer := NewHousingSync(client, repo)

	result := syncer.RunOnce(context.Background())

	if result.Error != "" {
		t.Fatalf("unexpected error: %s", result.Error)
	}
	if result.APICount != 2 {
		t.Errorf("expected api_count 2, got %d", result.APICount)
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

func TestRunOnce_SkipsBlockedHomeCodes(t *testing.T) {
	ts := newTestServer(`{
		"resultList": [
			{"homeCode": "20000594", "homeName": "test26", "supplyStatus": "02"},
			{"homeCode": "20000352", "homeName": "충정로역 어바니엘", "supplyStatus": "05"}
		]
	}`)
	defer ts.Close()

	repo := &mockHousingRepo{upsertUpdated: 1}
	client := NewHousingClient().WithHTTPClient(ts.Client()).WithListURL(ts.URL)
	syncer := NewHousingSync(client, repo)

	result := syncer.RunOnce(context.Background())

	if result.FetchedCount != 1 {
		t.Errorf("expected fetched 1 (blocked filtered), got %d", result.FetchedCount)
	}
	if result.Error != "" {
		t.Fatalf("unexpected error: %s", result.Error)
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

func newListAndImageServer(listJSON string, imageBytes []byte) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("atchFileId") != "" {
			w.Header().Set("Content-Type", "image/png")
			w.Write(imageBytes)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(listJSON))
	}))
}

func TestSyncImages_DownloadsAndStores(t *testing.T) {
	imgBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x01, 0x02}
	ts := newListAndImageServer(`{
		"resultList": [
			{"homeCode": "H001", "homeName": "A", "supplyStatus": "02", "fileId": "abc123", "fileSn": 1},
			{"homeCode": "H002", "homeName": "B", "supplyStatus": "03"}
		]
	}`, imgBytes)
	defer ts.Close()

	repo := &mockHousingRepo{upsertUpdated: 2}
	client := NewHousingClient().
		WithHTTPClient(ts.Client()).
		WithListURL(ts.URL).
		WithImageURL(ts.URL)
	syncer := NewHousingSync(client, repo)

	syncer.RunOnce(context.Background())

	// Only H001 has a fileId -> exactly one image stored.
	if len(repo.upsertedImages) != 1 {
		t.Fatalf("expected 1 image upserted, got %d", len(repo.upsertedImages))
	}
	got := repo.upsertedImages[0]
	if got.HomeCode != "H001" || got.FileID != "abc123" || got.FileSn != 1 {
		t.Errorf("unexpected image ref: %+v", got)
	}
	if string(got.Data) != string(imgBytes) {
		t.Errorf("image bytes mismatch")
	}
	if got.ContentType != "image/png" {
		t.Errorf("expected image/png, got %s", got.ContentType)
	}
	if got.ETag == "" {
		t.Error("expected non-empty etag")
	}
}

func TestSyncImages_SkipsUnchanged(t *testing.T) {
	ts := newListAndImageServer(`{
		"resultList": [
			{"homeCode": "H001", "homeName": "A", "supplyStatus": "02", "fileId": "abc123", "fileSn": 1}
		]
	}`, []byte{0x01})
	defer ts.Close()

	repo := &mockHousingRepo{
		upsertUpdated: 1,
		imageRefs: map[string]struct {
			fileID string
			fileSn int
		}{
			"H001": {fileID: "abc123", fileSn: 1},
		},
	}
	client := NewHousingClient().
		WithHTTPClient(ts.Client()).
		WithListURL(ts.URL).
		WithImageURL(ts.URL)
	syncer := NewHousingSync(client, repo)

	syncer.RunOnce(context.Background())

	if len(repo.upsertedImages) != 0 {
		t.Fatalf("expected 0 image upserts (unchanged ref), got %d", len(repo.upsertedImages))
	}
}

func TestRunOnce_FillsNewHousingDetails(t *testing.T) {
	listJSON := `{"resultList":[{"homeCode":"H001","homeName":"A","supplyStatus":"02"}]}`
	detailHTML := `<script>var xpos = "127.05"; var ypos = "37.58";</script><p>대표전화 : 02-1234</p>`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Query().Get("homeCode") != "" {
			w.Header().Set("Content-Type", "text/html; charset=UTF-8")
			w.Write([]byte(detailHTML))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(listJSON))
	}))
	defer ts.Close()

	repo := &mockHousingRepo{upsertNewCodes: []string{"H001"}}
	client := NewHousingClient().WithHTTPClient(ts.Client()).WithListURL(ts.URL).WithDetailURL(ts.URL)
	syncer := NewHousingSync(client, repo)

	syncer.RunOnce(context.Background())

	if len(repo.detailUpdates) != 1 || repo.detailUpdates[0] != "H001" {
		t.Fatalf("expected detail update for H001, got %v", repo.detailUpdates)
	}
}

func TestRunOnce_NoNewHousings_SkipsDetailFill(t *testing.T) {
	ts := newTestServer(`{"resultList":[{"homeCode":"H001","homeName":"A","supplyStatus":"02"}]}`)
	defer ts.Close()

	repo := &mockHousingRepo{upsertUpdated: 1} // no new codes
	client := NewHousingClient().WithHTTPClient(ts.Client()).WithListURL(ts.URL).WithDetailURL(ts.URL)
	syncer := NewHousingSync(client, repo)

	syncer.RunOnce(context.Background())

	if len(repo.detailUpdates) != 0 {
		t.Fatalf("expected no detail fills, got %v", repo.detailUpdates)
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
