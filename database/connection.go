package database

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	postgresDataSourceName = "postgres://postgres:CardCollector2025@localhost:5432/card_collectors_uk"
)

var (
	ConnectionPool *pgxpool.Pool
)

func InitialiseConnection(ctx context.Context) error {
	pool, err := pgxpool.New(ctx, postgresDataSourceName)
	if err != nil {
		return err
	}

	ConnectionPool = pool

	err = ConnectionPool.Ping(ctx)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	return nil
}

func CloseConnection() {
	if ConnectionPool != nil {
		ConnectionPool.Close()
	}

	ConnectionPool = nil
}
