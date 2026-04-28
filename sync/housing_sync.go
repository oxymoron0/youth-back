package sync

import (
	"context"
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

	updated, newCount, err := s.repo.UpsertFromListAPI(ctx, items)
	if err != nil {
		result.Error = err.Error()
		s.finalize(ctx, &result, startedAt)
		return result
	}

	result.UpdatedCount = updated
	result.NewCount = newCount
	s.finalize(ctx, &result, startedAt)
	return result
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
