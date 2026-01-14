package postgres

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sifan077/PowerURL/config"
)

const defaultDialTimeout = 5 * time.Second

// NewPool creates a pgx connection pool using the provided config and verifies connectivity.
func NewPool(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error) {
	host := cfg.Host
	if host == "" {
		host = "localhost"
	}
	port := cfg.Port
	if port == 0 {
		port = 5432
	}
	sslMode := cfg.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}

	connString := buildConnString(connParts{
		host:     host,
		port:     port,
		user:     cfg.User,
		password: cfg.Password,
		database: cfg.Database,
		sslMode:  sslMode,
	})

	poolCfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse config: %w", err)
	}

	dialCtx, cancel := context.WithTimeout(ctx, defaultDialTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(dialCtx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: create pool: %w", err)
	}

	pingCtx, pingCancel := context.WithTimeout(ctx, defaultDialTimeout)
	defer pingCancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}

	return pool, nil
}

type connParts struct {
	host     string
	port     int
	user     string
	password string
	database string
	sslMode  string
}

func buildConnString(parts connParts) string {
	user := url.PathEscape(parts.user)
	password := url.PathEscape(parts.password)
	database := url.PathEscape(parts.database)

	credentials := user
	if password != "" {
		credentials = fmt.Sprintf("%s:%s", user, password)
	}

	return fmt.Sprintf("postgres://%s@%s:%d/%s?sslmode=%s",
		credentials,
		parts.host,
		parts.port,
		database,
		parts.sslMode,
	)
}
