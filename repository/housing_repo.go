package repository

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"kr-metro-api/model"
)

type HousingRepository interface {
	List(ctx context.Context) ([]model.HousingListItem, error)
	GetByHomeCode(ctx context.Context, homeCode string) (*model.HousingDetail, error)
	NearbyStations(ctx context.Context, homeCode string, distanceMeters int) ([]model.NearbyStation, error)
	UpsertFromListAPI(ctx context.Context, items []model.HousingSyncItem) (updated, newCount int, err error)
	SaveSyncResult(ctx context.Context, result model.HousingSyncResult) error
	LatestSyncResult(ctx context.Context) (*model.HousingSyncResult, error)
	RecentSyncHistory(ctx context.Context, limit int) ([]model.HousingSyncResult, error)
}

type HousingRepo struct {
	pool *pgxpool.Pool
}

func NewHousingRepo(pool *pgxpool.Pool) *HousingRepo {
	return &HousingRepo{pool: pool}
}

func (r *HousingRepo) List(ctx context.Context) ([]model.HousingListItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT home_code, home_name, supply_status, address_gu, longitude, latitude
		FROM youth_housing.housings
		ORDER BY CASE supply_status
			WHEN '02' THEN 1
			WHEN '03' THEN 2
			WHEN '01' THEN 3
			WHEN '06' THEN 4
			WHEN '04' THEN 5
			WHEN '05' THEN 6
			WHEN '07' THEN 7
			ELSE 8
		END, housing_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.HousingListItem
	for rows.Next() {
		var h model.HousingListItem
		if err := rows.Scan(&h.HomeCode, &h.HomeName, &h.SupplyStatus,
			&h.AddressGu, &h.Longitude, &h.Latitude); err != nil {
			return nil, err
		}
		items = append(items, h)
	}
	if items == nil {
		items = []model.HousingListItem{}
	}
	return items, nil
}

func (r *HousingRepo) GetByHomeCode(ctx context.Context, homeCode string) (*model.HousingDetail, error) {
	var d model.HousingDetail
	err := r.pool.QueryRow(ctx, `
		SELECT housing_id, home_code, home_name, address, address_gu,
		       option_subway, supply_status, deposit_low, rental_low,
		       homepage_url, phone,
		       to_char(first_recruit_date, 'YYYY-MM-DD'),
		       to_char(move_in_date, 'YYYY-MM-DD'),
		       total_units, developer, constructor, latitude, longitude
		FROM youth_housing.housings
		WHERE home_code = $1
	`, homeCode).Scan(&d.HousingID, &d.HomeCode, &d.HomeName, &d.Address, &d.AddressGu,
		&d.OptionSubway, &d.SupplyStatus, &d.DepositLow, &d.RentalLow,
		&d.HomepageURL, &d.Phone, &d.FirstRecruitDate, &d.MoveInDate,
		&d.TotalUnits, &d.Developer, &d.Constructor, &d.Latitude, &d.Longitude)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

func (r *HousingRepo) NearbyStations(ctx context.Context, homeCode string, distanceMeters int) ([]model.NearbyStation, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT s.station_id, s.station_name,
		       string_agg(DISTINCT l.line_name, ', ' ORDER BY l.line_name),
		       round(ST_Distance(s.geom::geography, h.geom::geography)::numeric, 0),
		       s.latitude, s.longitude
		FROM youth_housing.housings h
		JOIN stations s ON ST_DWithin(s.geom::geography, h.geom::geography, $2)
		JOIN station_lines sl ON sl.station_id = s.station_id
		JOIN lines l ON l.line_id = sl.line_id
		WHERE h.home_code = $1 AND s.geom IS NOT NULL AND h.geom IS NOT NULL
		GROUP BY s.station_id, s.station_name, s.latitude, s.longitude, s.geom, h.geom
		ORDER BY ST_Distance(s.geom::geography, h.geom::geography)
	`, homeCode, distanceMeters)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stations []model.NearbyStation
	for rows.Next() {
		var ns model.NearbyStation
		if err := rows.Scan(&ns.StationID, &ns.StationName, &ns.LineNames,
			&ns.DistanceM, &ns.Latitude, &ns.Longitude); err != nil {
			return nil, err
		}
		stations = append(stations, ns)
	}
	if stations == nil {
		stations = []model.NearbyStation{}
	}
	return stations, nil
}

func (r *HousingRepo) UpsertFromListAPI(ctx context.Context, items []model.HousingSyncItem) (updated, newCount int, err error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const upsertSQL = `
		INSERT INTO youth_housing.housings
			(home_code, home_name, address, address_gu, option_subway,
			 supply_status, deposit_low, rental_low, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (home_code) DO UPDATE SET
			home_name     = EXCLUDED.home_name,
			supply_status = EXCLUDED.supply_status,
			deposit_low   = EXCLUDED.deposit_low,
			rental_low    = EXCLUDED.rental_low,
			updated_at    = NOW()
		RETURNING (xmax = 0) AS is_insert
	`

	batch := &pgx.Batch{}
	for _, item := range items {
		deposit := parseMoney(item.DepositLow)
		rental := parseMoney(item.RentalLow)
		batch.Queue(upsertSQL,
			item.HomeCode, item.HomeName, item.Address, item.AddressGu,
			item.OptionSubway, item.SupplyStatus, deposit, rental,
		)
	}

	br := tx.SendBatch(ctx, batch)
	for range items {
		var isInsert bool
		if err := br.QueryRow().Scan(&isInsert); err != nil {
			br.Close()
			return 0, 0, fmt.Errorf("upsert scan: %w", err)
		}
		if isInsert {
			newCount++
		} else {
			updated++
		}
	}
	if err := br.Close(); err != nil {
		return 0, 0, fmt.Errorf("batch close: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, fmt.Errorf("commit: %w", err)
	}

	return updated, newCount, nil
}

func (r *HousingRepo) SaveSyncResult(ctx context.Context, result model.HousingSyncResult) error {
	var errVal *string
	if result.Error != "" {
		e := result.Error
		errVal = &e
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO youth_housing.sync_history
			(started_at, completed_at, duration_ms, fetched_count, updated_count, new_count, error)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, result.StartedAt, result.CompletedAt, result.DurationMs,
		result.FetchedCount, result.UpdatedCount, result.NewCount, errVal)
	if err != nil {
		return fmt.Errorf("insert sync_history: %w", err)
	}
	return nil
}

func (r *HousingRepo) LatestSyncResult(ctx context.Context) (*model.HousingSyncResult, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT started_at, completed_at, duration_ms,
		       fetched_count, updated_count, new_count, error
		FROM youth_housing.sync_history
		ORDER BY started_at DESC
		LIMIT 1
	`)
	res, err := scanSyncResult(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query latest sync_history: %w", err)
	}
	return &res, nil
}

func (r *HousingRepo) RecentSyncHistory(ctx context.Context, limit int) ([]model.HousingSyncResult, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `
		SELECT started_at, completed_at, duration_ms,
		       fetched_count, updated_count, new_count, error
		FROM youth_housing.sync_history
		ORDER BY started_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query sync_history: %w", err)
	}
	defer rows.Close()

	results := make([]model.HousingSyncResult, 0, limit)
	for rows.Next() {
		res, err := scanSyncResult(rows)
		if err != nil {
			return nil, fmt.Errorf("scan sync_history: %w", err)
		}
		results = append(results, res)
	}
	return results, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSyncResult(row rowScanner) (model.HousingSyncResult, error) {
	var res model.HousingSyncResult
	var errVal *string
	if err := row.Scan(&res.StartedAt, &res.CompletedAt, &res.DurationMs,
		&res.FetchedCount, &res.UpdatedCount, &res.NewCount, &errVal); err != nil {
		return res, err
	}
	if errVal != nil {
		res.Error = *errVal
	}
	res.Duration = (time.Duration(res.DurationMs) * time.Millisecond).String()
	return res, nil
}

// parseMoney handles the Seoul API's mixed int/string money fields.
func parseMoney(v any) *int64 {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case float64:
		n := int64(math.Round(val))
		return &n
	case string:
		if val == "" {
			return nil
		}
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil
		}
		return &n
	default:
		return nil
	}
}
