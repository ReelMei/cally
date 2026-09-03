package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

const migrationsSchema = `
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(64) PRIMARY KEY,
    display_name VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS rooms (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    host_id VARCHAR(64) NOT NULL,
    max_participants INT NOT NULL DEFAULT 6,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_rooms_host_id ON rooms(host_id);
CREATE INDEX IF NOT EXISTS idx_rooms_is_active ON rooms(is_active);

CREATE TABLE IF NOT EXISTS room_participants (
    id SERIAL PRIMARY KEY,
    room_id VARCHAR(64) NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    user_id VARCHAR(64) NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    is_host BOOLEAN NOT NULL DEFAULT FALSE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    left_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_participants_room_id ON room_participants(room_id);
CREATE INDEX IF NOT EXISTS idx_participants_user_id ON room_participants(user_id);

CREATE TABLE IF NOT EXISTS call_logs (
    id VARCHAR(64) PRIMARY KEY,
    room_id VARCHAR(64) NOT NULL,
    user_id VARCHAR(64),
    event_type VARCHAR(50) NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_call_logs_room_id ON call_logs(room_id);
CREATE INDEX IF NOT EXISTS idx_call_logs_event_type ON call_logs(event_type);
`

func RunMigrations(ctx context.Context, db *sql.DB) error {
	slog.Info("running PostgreSQL database schema migrations...")

	ctxTimeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	_, err := db.ExecContext(ctxTimeout, migrationsSchema)
	if err != nil {
		return fmt.Errorf("failed executing database migrations: %w", err)
	}

	slog.Info("PostgreSQL database migrations completed successfully")
	return nil
}
