package repository

import (
	"context"

	"cally/internal/models"
)

type RoomRepository interface {
	CreateRoom(ctx context.Context, room *models.RoomInfo) error
	GetRoomByID(ctx context.Context, roomID string) (*models.RoomInfo, error)
	ListActiveRooms(ctx context.Context) ([]models.RoomInfo, error)
	CloseRoom(ctx context.Context, roomID string) error
	DeleteRoom(ctx context.Context, roomID string) error
}

type UserRepository interface {
	UpsertUser(ctx context.Context, userID, displayName string) error
	GetUserByID(ctx context.Context, userID string) (*models.PeerInfo, error)
}

type ParticipantRepository interface {
	AddParticipant(ctx context.Context, roomID, userID, displayName string, isHost bool) error
	RemoveParticipant(ctx context.Context, roomID, userID string) error
}

type CallLogRepository interface {
	LogEvent(ctx context.Context, eventID, roomID, userID, eventType string, metadata interface{}) error
}

type Repositories struct {
	Rooms        RoomRepository
	Users        UserRepository
	Participants ParticipantRepository
	CallLogs     CallLogRepository
}
