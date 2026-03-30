package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	if dsn == "" {
		return nil, fmt.Errorf("KR_METRO_DSN not set")
	}
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse DSN: %w", err)
	}
	config.MaxConns = 10
	config.MinConns = 2
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}
