package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"cally/internal/models"
)

// PostgreSQL implementation of RoomRepository
type PostgresRoomRepository struct {
	db *sql.DB
}

func NewPostgresRoomRepository(db *sql.DB) *PostgresRoomRepository {
	return &PostgresRoomRepository{db: db}
}

func (r *PostgresRoomRepository) CreateRoom(ctx context.Context, room *models.RoomInfo) error {
	query := `
		INSERT INTO rooms (id, name, host_id, max_participants, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, TRUE, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			host_id = EXCLUDED.host_id,
			max_participants = EXCLUDED.max_participants,
			is_active = TRUE,
			updated_at = NOW();
	`
	_, err := r.db.ExecContext(ctx, query, room.ID, room.Name, room.HostID, room.MaxPeers)
	if err != nil {
		return fmt.Errorf("postgres CreateRoom query error: %w", err)
	}
	return nil
}

func (r *PostgresRoomRepository) GetRoomByID(ctx context.Context, roomID string) (*models.RoomInfo, error) {
	query := `
		SELECT id, name, host_id, max_participants, created_at
		FROM rooms
		WHERE id = $1 AND is_active = TRUE;
	`
	var info models.RoomInfo
	var createdAt time.Time

	err := r.db.QueryRowContext(ctx, query, roomID).Scan(
		&info.ID,
		&info.Name,
		&info.HostID,
		&info.MaxPeers,
		&createdAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrRoomNotFound
		}
		return nil, fmt.Errorf("postgres GetRoomByID query error: %w", err)
	}
	info.CreatedAt = createdAt.UnixMilli()

	// Load active participants
	peersQuery := `
		SELECT user_id, display_name, is_host, joined_at
		FROM room_participants
		WHERE room_id = $1 AND left_at IS NULL;
	`
	rows, err := r.db.QueryContext(ctx, peersQuery, roomID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var p models.PeerInfo
			var joinedAt time.Time
			if err := rows.Scan(&p.ID, &p.DisplayName, &p.IsHost, &joinedAt); err == nil {
				p.JoinedAt = joinedAt.UnixMilli()
				info.Peers = append(info.Peers, p)
			}
		}
	}

	return &info, nil
}

func (r *PostgresRoomRepository) ListActiveRooms(ctx context.Context) ([]models.RoomInfo, error) {
	query := `
		SELECT id, name, host_id, max_participants, created_at
		FROM rooms
		WHERE is_active = TRUE
		ORDER BY created_at DESC;
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("postgres ListActiveRooms query error: %w", err)
	}
	defer rows.Close()

	var rooms []models.RoomInfo
	for rows.Next() {
		var info models.RoomInfo
		var createdAt time.Time
		if err := rows.Scan(&info.ID, &info.Name, &info.HostID, &info.MaxPeers, &createdAt); err == nil {
			info.CreatedAt = createdAt.UnixMilli()
			rooms = append(rooms, info)
		}
	}
	return rooms, nil
}

func (r *PostgresRoomRepository) CloseRoom(ctx context.Context, roomID string) error {
	query := `
		UPDATE rooms
		SET is_active = FALSE, ended_at = NOW(), updated_at = NOW()
		WHERE id = $1;
	`
	_, err := r.db.ExecContext(ctx, query, roomID)
	if err != nil {
		return fmt.Errorf("postgres CloseRoom error: %w", err)
	}

	// Mark remaining active participants as left
	partQuery := `
		UPDATE room_participants
		SET left_at = NOW()
		WHERE room_id = $1 AND left_at IS NULL;
	`
	_, _ = r.db.ExecContext(ctx, partQuery, roomID)
	return nil
}

func (r *PostgresRoomRepository) DeleteRoom(ctx context.Context, roomID string) error {
	query := `DELETE FROM rooms WHERE id = $1;`
	res, err := r.db.ExecContext(ctx, query, roomID)
	if err != nil {
		return fmt.Errorf("postgres DeleteRoom error: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return models.ErrRoomNotFound
	}
	return nil
}

// PostgreSQL implementation of UserRepository
type PostgresUserRepository struct {
	db *sql.DB
}

func NewPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) UpsertUser(ctx context.Context, userID, displayName string) error {
	query := `
		INSERT INTO users (id, display_name, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			updated_at = NOW();
	`
	_, err := r.db.ExecContext(ctx, query, userID, displayName)
	if err != nil {
		return fmt.Errorf("postgres UpsertUser error: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) GetUserByID(ctx context.Context, userID string) (*models.PeerInfo, error) {
	query := `SELECT id, display_name, created_at FROM users WHERE id = $1;`
	var info models.PeerInfo
	var createdAt time.Time
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&info.ID, &info.DisplayName, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrPeerNotFound
		}
		return nil, err
	}
	info.JoinedAt = createdAt.UnixMilli()
	return &info, nil
}

// PostgreSQL implementation of ParticipantRepository
type PostgresParticipantRepository struct {
	db *sql.DB
}

func NewPostgresParticipantRepository(db *sql.DB) *PostgresParticipantRepository {
	return &PostgresParticipantRepository{db: db}
}

func (r *PostgresParticipantRepository) AddParticipant(ctx context.Context, roomID, userID, displayName string, isHost bool) error {
	query := `
		INSERT INTO room_participants (room_id, user_id, display_name, is_host, joined_at)
		VALUES ($1, $2, $3, $4, NOW());
	`
	_, err := r.db.ExecContext(ctx, query, roomID, userID, displayName, isHost)
	if err != nil {
		return fmt.Errorf("postgres AddParticipant error: %w", err)
	}
	return nil
}

func (r *PostgresParticipantRepository) RemoveParticipant(ctx context.Context, roomID, userID string) error {
	query := `
		UPDATE room_participants
		SET left_at = NOW()
		WHERE room_id = $1 AND user_id = $2 AND left_at IS NULL;
	`
	_, err := r.db.ExecContext(ctx, query, roomID, userID)
	if err != nil {
		return fmt.Errorf("postgres RemoveParticipant error: %w", err)
	}
	return nil
}

// PostgreSQL implementation of CallLogRepository
type PostgresCallLogRepository struct {
	db *sql.DB
}

func NewPostgresCallLogRepository(db *sql.DB) *PostgresCallLogRepository {
	return &PostgresCallLogRepository{db: db}
}

func (r *PostgresCallLogRepository) LogEvent(ctx context.Context, eventID, roomID, userID, eventType string, metadata interface{}) error {
	var metaJSON []byte
	if metadata != nil {
		metaJSON, _ = json.Marshal(metadata)
	}

	query := `
		INSERT INTO call_logs (id, room_id, user_id, event_type, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW());
	`
	_, err := r.db.ExecContext(ctx, query, eventID, roomID, userID, eventType, string(metaJSON))
	if err != nil {
		return fmt.Errorf("postgres LogEvent error: %w", err)
	}
	return nil
}
