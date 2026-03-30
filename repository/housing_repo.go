package repository

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"kr-metro-api/model"
)

type HousingRepository interface {
	List(ctx context.Context) ([]model.HousingListItem, error)
	GetByHomeCode(ctx context.Context, homeCode string) (*model.HousingDetail, error)
	NearbyStations(ctx context.Context, homeCode string, distanceMeters int) ([]model.NearbyStation, error)
	UpsertFromListAPI(ctx context.Context, items []model.HousingSyncItem) (updated, newCount int, err error)
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
