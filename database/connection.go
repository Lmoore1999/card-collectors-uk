package database

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"os"
	"time"
)

const (
	postgresDataSourceName = "postgres://postgres:CardCollector2025@localhost:5432/card_collectors_uk"
)

var (
	ConnectionPool *pgxpool.Pool
)

func InitialiseConnection(ctx context.Context) error {
	err := godotenv.Load(".env")
	if err != nil {
		return fmt.Errorf("error loading .env file: %s", err.Error())
	}

	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return fmt.Errorf("error parsing connection string: %w", err)
	}

	// Optional: Configure max connections for your pool
	config.MaxConns = 1000                   // max open connections
	config.MinConns = 10                     // minimum idle connections to keep
	config.MaxConnLifetime = 5 * time.Minute // recycle connections after 5m

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return fmt.Errorf("error creating connection pool: %w", err)
	}

	var now time.Time
	err = pool.QueryRow(ctx, "SELECT NOW()").Scan(&now)
	if err != nil {
		pool.Close()
		return fmt.Errorf("error querying db time: %w", err)
	}

	fmt.Printf("Database initialised DB Server Time: %s\n", now)

	ConnectionPool = pool
	return nil
}

func CloseConnection() {
	if ConnectionPool != nil {
		ConnectionPool.Close()
	}

	ConnectionPool = nil
}

func executeFunction(ctx context.Context, functionName string) (pgx.Rows, error) {
	if ConnectionPool == nil {
		return nil, errors.New("database connection pool is not initialized")
	}

	rows, err := ConnectionPool.Query(ctx, fmt.Sprintf("SELECT * FROM public.%s()", functionName))
	if err != nil {
		return nil, err
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return rows, nil
}
