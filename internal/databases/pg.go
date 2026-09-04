package databases

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PgDatabase struct {
	Pool *pgxpool.Pool
}

func (s *PgDatabase) CreatePgDatabase(host string, port int32, user, password, dbname string) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s", user, password, host, port, dbname)

	poolConfig, errConfig := pgxpool.ParseConfig(dsn)
	if errConfig != nil {
		log.Fatalf("Error parsing pg config: %v", errConfig)
	}

	poolConfig.ConnConfig.ConnectTimeout = 10 * time.Second
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 1
	poolConfig.MaxConnLifetime = 10 * time.Minute
	poolConfig.MaxConnIdleTime = time.Minute
	poolConfig.HealthCheckPeriod = time.Minute

	pool, errPool := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if errPool != nil {
		log.Fatalf("Error creating pg pool: %v", errPool)
	}
	s.Pool = pool
}
