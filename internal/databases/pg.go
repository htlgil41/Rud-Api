package databases

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PgDatabase struct {
	Pool *pgxpool.Pool
}

func (s *PgDatabase) CreatePgDatabase() {
	pool, errPool := pgxpool.NewWithConfig(context.Background(), &pgxpool.Config{
		ConnConfig: &pgx.ConnConfig{
			Config: pgconn.Config{
				Host:           "",
				Port:           5432,
				User:           "",
				Password:       "",
				Database:       "",
				ConnectTimeout: 10 * time.Second,
			},
		},
		MaxConns:          10,
		MinConns:          1,
		MaxConnLifetime:   10 * time.Minute,
		MaxConnIdleTime:   time.Minute,
		HealthCheckPeriod: time.Minute,
	})
	if errPool != nil {
		return
	}
	s.Pool = pool
}
