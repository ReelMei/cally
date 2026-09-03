package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"cally/internal/config"

	_ "github.com/lib/pq"
)

type DB struct {
	SQLDB *sql.DB
}

func Connect(ctx context.Context, cfg *config.Config) (*DB, error) {
	if !cfg.DBEnable {
		slog.Info("database integration disabled by configuration")
		return nil, nil
	}

	dsn := cfg.GetDSN()
	slog.Info("connecting to PostgreSQL database...", "host", cfg.DBHost, "port", cfg.DBPort, "db", cfg.DBName)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgresql database connection: %w", err)
	}

	// Performance Optimization: Connection Pool Tuning
	db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)
	db.SetConnMaxLifetime(15 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("postgresql ping failed: %w", err)
	}

	slog.Info("PostgreSQL connection established successfully",
		"maxOpenConns", cfg.DBMaxOpenConns,
		"maxIdleConns", cfg.DBMaxIdleConns,
	)

	// Run DDL migrations
	if err := RunMigrations(ctx, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrations error: %w", err)
	}

	return &DB{SQLDB: db}, nil
}

func (d *DB) Close() error {
	if d.SQLDB != nil {
		slog.Info("closing PostgreSQL database connection pool")
		return d.SQLDB.Close()
	}
	return nil
}
