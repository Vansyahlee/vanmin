package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Open(ctx context.Context) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig("")
	if err != nil {
		return nil, errors.New("konfigurasi PostgreSQL tidak valid; periksa variabel PG di environment")
	}

	poolConfig.ConnConfig.ConnectTimeout = 5 * time.Second
	poolConfig.MaxConns = 5

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat pool PostgreSQL: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("gagal terhubung ke PostgreSQL: %w", err)
	}

	return pool, nil
}
