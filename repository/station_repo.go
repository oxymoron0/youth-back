package repository

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"kr-metro-api/model"
)

type StationRepository interface {
	List(ctx context.Context, page, perPage int) ([]model.Station, int, error)
	ListGeoJSON(ctx context.Context, page, perPage int) (json.RawMessage, int, error)
	ListGeoJSONAll(ctx context.Context, lineID *int) (json.RawMessage, error)
	GetByID(ctx context.Context, id int) (*model.StationDetail, error)
	Search(ctx context.Context, query string, limit int) ([]model.SearchResult, error)
	Nearby(ctx context.Context, lon, lat float64, radius, limit int) (json.RawMessage, error)
}

type StationRepo struct {
	pool *pgxpool.Pool
}

func NewStationRepo(pool *pgxpool.Pool) *StationRepo {
	return &StationRepo{pool: pool}
}

func (r *StationRepo) List(ctx context.Context, page, perPage int) ([]model.Station, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM stations").Scan(&total); err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * perPage
	rows, err := r.pool.Query(ctx, `
		SELECT station_id, station_code, station_name, station_name_en,
		       latitude, longitude, is_transfer
		FROM stations
		ORDER BY station_id
		LIMIT $1 OFFSET $2
	`, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var stations []model.Station
	for rows.Next() {
		var s model.Station
		if err := rows.Scan(&s.StationID, &s.StationCode, &s.StationName,
			&s.StationNameEn, &s.Latitude, &s.Longitude, &s.IsTransfer); err != nil {
			return nil, 0, err
		}
		stations = append(stations, s)
	}
	if stations == nil {
		stations = []model.Station{}
	}
	return stations, total, nil
}

func (r *StationRepo) ListGeoJSON(ctx context.Context, page, perPage int) (json.RawMessage, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM stations WHERE geom IS NOT NULL").Scan(&total); err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * perPage
	var result json.RawMessage
	err := r.pool.QueryRow(ctx, `
		SELECT json_build_object(
			'type', 'FeatureCollection',
			'features', COALESCE(json_agg(sub.feature), '[]'::json)
		)
		FROM (
			SELECT feature
			FROM v_stations_geojson
			ORDER BY station_id
			LIMIT $1 OFFSET $2
		) sub
	`, perPage, offset).Scan(&result)
	if err != nil {
		return nil, 0, err
	}
	return result, total, nil
}

func (r *StationRepo) ListGeoJSONAll(ctx context.Context, lineID *int) (json.RawMessage, error) {
	var result json.RawMessage
	var err error
	if lineID != nil {
		err = r.pool.QueryRow(ctx, `SELECT fn_stations_fc($1)`, *lineID).Scan(&result)
	} else {
		err = r.pool.QueryRow(ctx, `SELECT fn_stations_fc()`).Scan(&result)
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *StationRepo) GetByID(ctx context.Context, id int) (*model.StationDetail, error) {
	var d model.StationDetail
	err := r.pool.QueryRow(ctx, `
		SELECT station_id, station_code, station_name, station_name_en, station_name_cn,
		       address, phone, latitude, longitude, is_transfer
		FROM stations WHERE station_id = $1
	`, id).Scan(&d.StationID, &d.StationCode, &d.StationName, &d.StationNameEn,
		&d.StationNameCn, &d.Address, &d.Phone, &d.Latitude, &d.Longitude, &d.IsTransfer)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Lines
	lrows, err := r.pool.Query(ctx, `
		SELECT l.line_id, l.line_code, l.line_name, l.line_color
		FROM station_lines sl
		JOIN lines l ON l.line_id = sl.line_id
		WHERE sl.station_id = $1
		ORDER BY l.line_name
	`, id)
	if err != nil {
		return nil, err
	}
	defer lrows.Close()
	for lrows.Next() {
		var sl model.StationLine
		if err := lrows.Scan(&sl.LineID, &sl.LineCode, &sl.LineName, &sl.LineColor); err != nil {
			return nil, err
		}
		d.Lines = append(d.Lines, sl)
	}
	if d.Lines == nil {
		d.Lines = []model.StationLine{}
	}

	// Transfers
	trows, err := r.pool.Query(ctx, `
		SELECT t.transfer_id, t.to_station_id, ts.station_name,
		       t.from_line_id, fl.line_name,
		       t.to_line_id, tl.line_name,
		       t.transfer_time
		FROM transfers t
		JOIN stations ts ON ts.station_id = t.to_station_id
		JOIN lines fl ON fl.line_id = t.from_line_id
		JOIN lines tl ON tl.line_id = t.to_line_id
		WHERE t.from_station_id = $1
		ORDER BY tl.line_name
	`, id)
	if err != nil {
		return nil, err
	}
	defer trows.Close()
	for trows.Next() {
		var st model.StationTransfer
		if err := trows.Scan(&st.TransferID, &st.ToStationID, &st.ToStation,
			&st.FromLineID, &st.FromLineName, &st.ToLineID, &st.ToLineName,
			&st.TransferTime); err != nil {
			return nil, err
		}
		d.Transfers = append(d.Transfers, st)
	}
	if d.Transfers == nil {
		d.Transfers = []model.StationTransfer{}
	}

	// Exits
	erows, err := r.pool.Query(ctx, `
		SELECT exit_id, exit_number, exit_name, latitude, longitude, is_accessible
		FROM station_exits
		WHERE station_id = $1
		ORDER BY exit_number
	`, id)
	if err != nil {
		return nil, err
	}
	defer erows.Close()
	for erows.Next() {
		var se model.StationExit
		if err := erows.Scan(&se.ExitID, &se.ExitNumber, &se.ExitName,
			&se.Latitude, &se.Longitude, &se.IsAccessible); err != nil {
			return nil, err
		}
		d.Exits = append(d.Exits, se)
	}
	if d.Exits == nil {
		d.Exits = []model.StationExit{}
	}

	return &d, nil
}

func (r *StationRepo) Search(ctx context.Context, query string, limit int) ([]model.SearchResult, error) {
	rows, err := r.pool.Query(ctx, `SELECT * FROM search_stations($1, $2)`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.SearchResult
	for rows.Next() {
		var sr model.SearchResult
		if err := rows.Scan(&sr.StationID, &sr.StationCode, &sr.StationName,
			&sr.StationNameEn, &sr.Latitude, &sr.Longitude, &sr.IsTransfer,
			&sr.SimilarityScore, &sr.Lines); err != nil {
			return nil, err
		}
		results = append(results, sr)
	}
	if results == nil {
		results = []model.SearchResult{}
	}
	return results, nil
}

func (r *StationRepo) Nearby(ctx context.Context, lon, lat float64, radius, limit int) (json.RawMessage, error) {
	var result json.RawMessage
	err := r.pool.QueryRow(ctx,
		`SELECT fn_nearby_stations($1, $2, $3, $4)`,
		lon, lat, radius, limit,
	).Scan(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
