package sync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"sync/atomic"
	"time"

	"kr-metro-api/model"
	"kr-metro-api/repository"
)

const defaultInterval = 60 * time.Minute

// blockedHomeCodes lists upstream API records to drop on sync (test/dummy data).
var blockedHomeCodes = map[string]struct{}{
	"20000594": {}, // home_name="test26", upstream test record
}

type HousingSync struct {
	client     *HousingClient
	repo       repository.HousingRepository
	interval   time.Duration
	lastResult atomic.Pointer[model.HousingSyncResult]
	logger     *slog.Logger
}

type Option func(*HousingSync)

func WithInterval(d time.Duration) Option {
	return func(s *HousingSync) {
		if d > 0 {
			s.interval = d
		}
	}
}

func WithLogger(l *slog.Logger) Option {
	return func(s *HousingSync) {
		s.logger = l
	}
}

func NewHousingSync(client *HousingClient, repo repository.HousingRepository, opts ...Option) *HousingSync {
	s := &HousingSync{
		client:   client,
		repo:     repo,
		interval: defaultInterval,
		logger:   slog.Default(),
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Start runs the sync loop: immediate first run, then on interval.
// Blocks until ctx is cancelled.
func (s *HousingSync) Start(ctx context.Context) {
	s.logger.Info("housing sync started", "interval", s.interval)

	// Immediate first run
	result := s.RunOnce(ctx)
	s.logResult(result)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("housing sync stopped")
			return
		case <-ticker.C:
			result := s.RunOnce(ctx)
			s.logResult(result)
		}
	}
}

// RunOnce performs a single sync cycle: fetch from API and upsert into DB.
func (s *HousingSync) RunOnce(ctx context.Context) model.HousingSyncResult {
	startedAt := time.Now()
	result := model.HousingSyncResult{StartedAt: startedAt}

	items, err := s.client.FetchList(ctx)
	if err != nil {
		result.Error = err.Error()
		s.finalize(ctx, &result, startedAt)
		return result
	}

	// api_count = number of housings the API returned this cycle (used to
	// detect an increase vs the previous sync).
	result.APICount = len(items)

	kept := items[:0]
	for _, item := range items {
		if _, blocked := blockedHomeCodes[item.HomeCode]; blocked {
			s.logger.Info("skipped blocked housing",
				"home_code", item.HomeCode, "home_name", item.HomeName)
			continue
		}
		kept = append(kept, item)
	}
	items = kept
	result.FetchedCount = len(items)

	updated, newHomeCodes, err := s.repo.UpsertFromListAPI(ctx, items)
	if err != nil {
		result.Error = err.Error()
		s.finalize(ctx, &result, startedAt)
		return result
	}

	result.UpdatedCount = updated
	result.NewCount = len(newHomeCodes)

	// Decide whether to run the detail scrape: compare the API housing count
	// with the previous sync. When it increased, newly inserted housings exist
	// and their detail fields (coords, phone, dates, ...) need filling.
	prevCount := s.previousAPICount(ctx)
	if len(newHomeCodes) > 0 {
		s.logger.Info("housing count increased; filling new housing details",
			"prev_api_count", prevCount, "api_count", result.APICount,
			"new", len(newHomeCodes))
		s.fillDetails(ctx, newHomeCodes)
	}

	// Best-effort: download representative images. Image failures must not
	// fail the sync cycle (images are auxiliary to the housing data).
	s.syncImages(ctx, items)

	s.finalize(ctx, &result, startedAt)
	return result
}

// previousAPICount returns the api_count recorded by the most recent prior sync
// (0 if none / unavailable).
func (s *HousingSync) previousAPICount(ctx context.Context) int {
	prev, err := s.repo.LatestSyncResult(ctx)
	if err != nil || prev == nil {
		return 0
	}
	return prev.APICount
}

// fillDetails scrapes each newly added housing's detail page and fills the
// fields the list API does not provide. Best-effort: per-housing failures are
// logged and skipped, never failing the sync cycle.
func (s *HousingSync) fillDetails(ctx context.Context, homeCodes []string) {
	var filled, failed int
	for _, code := range homeCodes {
		if ctx.Err() != nil {
			return
		}
		detail, err := s.client.FetchDetail(ctx, code)
		if err != nil {
			failed++
			s.logger.Warn("housing detail fetch failed", "home_code", code, "error", err)
			continue
		}
		if err := s.repo.UpdateHousingDetail(ctx, code, detail); err != nil {
			failed++
			s.logger.Warn("housing detail store failed", "home_code", code, "error", err)
			continue
		}
		filled++
	}
	if filled > 0 || failed > 0 {
		s.logger.Info("new housing details filled", "filled", filled, "failed", failed)
	}
}

// syncImages downloads each housing's representative image (from the list API's
// fileId/fileSn) and stores the bytes, skipping any image whose source reference
// is unchanged since the last fetch.
func (s *HousingSync) syncImages(ctx context.Context, items []model.HousingSyncItem) {
	var fetched, skipped, failed int
	for _, item := range items {
		if ctx.Err() != nil {
			return
		}
		if item.FileID == "" {
			continue
		}
		fileSn := item.FileSn
		if fileSn <= 0 {
			fileSn = 1
		}

		if curID, curSn, ok, err := s.repo.ImageRef(ctx, item.HomeCode); err == nil &&
			ok && curID == item.FileID && curSn == fileSn {
			skipped++
			continue
		}

		data, contentType, err := s.client.FetchImage(ctx, item.FileID, fileSn)
		if err != nil {
			failed++
			s.logger.Warn("housing image fetch failed",
				"home_code", item.HomeCode, "file_id", item.FileID, "error", err)
			continue
		}

		sum := sha256.Sum256(data)
		img := model.HousingImage{
			HomeCode:    item.HomeCode,
			FileID:      item.FileID,
			FileSn:      fileSn,
			ContentType: contentType,
			ETag:        hex.EncodeToString(sum[:]),
			Data:        data,
		}
		if err := s.repo.UpsertImage(ctx, img); err != nil {
			failed++
			s.logger.Warn("housing image store failed",
				"home_code", item.HomeCode, "error", err)
			continue
		}
		fetched++
	}
	if fetched > 0 || failed > 0 {
		s.logger.Info("housing images synced",
			"fetched", fetched, "skipped", skipped, "failed", failed)
	}
}

func (s *HousingSync) finalize(ctx context.Context, result *model.HousingSyncResult, startedAt time.Time) {
	elapsed := time.Since(startedAt)
	result.CompletedAt = time.Now()
	result.Duration = elapsed.String()
	result.DurationMs = elapsed.Milliseconds()
	s.lastResult.Store(result)

	// Persist to DB; failure must not break the sync.
	if err := s.repo.SaveSyncResult(ctx, *result); err != nil {
		s.logger.Error("save sync_history failed", "error", err)
	}
}

// LastResult returns the most recent sync result.
func (s *HousingSync) LastResult() *model.HousingSyncResult {
	return s.lastResult.Load()
}

func (s *HousingSync) logResult(r model.HousingSyncResult) {
	if r.Error != "" {
		s.logger.Error("housing sync failed",
			"error", r.Error,
			"duration", r.Duration,
		)
		return
	}
	s.logger.Info("housing sync completed",
		"fetched", r.FetchedCount,
		"updated", r.UpdatedCount,
		"new", r.NewCount,
		"duration", r.Duration,
	)
}
