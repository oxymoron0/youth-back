package repository

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"kr-metro-api/model"
)

type LineRepository interface {
	List(ctx context.Context) ([]model.Line, error)
	ListStations(ctx context.Context, lineID int) (json.RawMessage, error)
	GetGeometry(ctx context.Context, lineID int) (json.RawMessage, error)
}

type LineRepo struct {
	pool *pgxpool.Pool
}

func NewLineRepo(pool *pgxpool.Pool) *LineRepo {
	return &LineRepo{pool: pool}
}

func (r *LineRepo) List(ctx context.Context) ([]model.Line, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT line_id, line_code, line_name, line_color,
		       start_station, end_station, opened_date::text,
		       line_length_km, operator_name, station_count
		FROM v_lines_summary
		ORDER BY line_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []model.Line
	for rows.Next() {
		var l model.Line
		if err := rows.Scan(&l.LineID, &l.LineCode, &l.LineName, &l.LineColor,
			&l.StartStation, &l.EndStation, &l.OpenedDate,
			&l.LineLengthKm, &l.OperatorName, &l.StationCount); err != nil {
			return nil, err
		}
		lines = append(lines, l)
	}
	if lines == nil {
		lines = []model.Line{}
	}
	return lines, nil
}

func (r *LineRepo) ListStations(ctx context.Context, lineID int) (json.RawMessage, error) {
	var result json.RawMessage
	err := r.pool.QueryRow(ctx, `SELECT fn_stations_fc($1)`, lineID).Scan(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *LineRepo) GetGeometry(ctx context.Context, lineID int) (json.RawMessage, error) {
	var result json.RawMessage
	err := r.pool.QueryRow(ctx, `SELECT fn_line_geometry($1)`, lineID).Scan(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
