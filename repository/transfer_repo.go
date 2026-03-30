package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"kr-metro-api/model"
)

type TransferRepository interface {
	GetByStation(ctx context.Context, stationID int) ([]model.Transfer, error)
}

type TransferRepo struct {
	pool *pgxpool.Pool
}

func NewTransferRepo(pool *pgxpool.Pool) *TransferRepo {
	return &TransferRepo{pool: pool}
}

func (r *TransferRepo) GetByStation(ctx context.Context, stationID int) ([]model.Transfer, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT t.transfer_id,
		       t.from_station_id, fs.station_name,
		       t.from_line_id, fl.line_name,
		       t.to_station_id, ts.station_name,
		       t.to_line_id, tl.line_name,
		       t.transfer_time
		FROM transfers t
		JOIN stations fs ON fs.station_id = t.from_station_id
		JOIN stations ts ON ts.station_id = t.to_station_id
		JOIN lines fl ON fl.line_id = t.from_line_id
		JOIN lines tl ON tl.line_id = t.to_line_id
		WHERE t.from_station_id = $1 OR t.to_station_id = $1
		ORDER BY fl.line_name, tl.line_name
	`, stationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transfers []model.Transfer
	for rows.Next() {
		var t model.Transfer
		if err := rows.Scan(&t.TransferID,
			&t.FromStationID, &t.FromStation,
			&t.FromLineID, &t.FromLineName,
			&t.ToStationID, &t.ToStation,
			&t.ToLineID, &t.ToLineName,
			&t.TransferTime); err != nil {
			return nil, err
		}
		transfers = append(transfers, t)
	}
	if transfers == nil {
		transfers = []model.Transfer{}
	}
	return transfers, nil
}
