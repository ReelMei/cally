package repository

import (
	"context"
	"sync"
	"time"

	"cally/internal/models"
)

type MemoryRoomRepository struct {
	rooms map[string]*models.RoomInfo
	mu    sync.RWMutex
}

func NewMemoryRoomRepository() *MemoryRoomRepository {
	return &MemoryRoomRepository{
		rooms: make(map[string]*models.RoomInfo),
	}
}

func (r *MemoryRoomRepository) CreateRoom(ctx context.Context, room *models.RoomInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rooms[room.ID] = room
	return nil
}

func (r *MemoryRoomRepository) GetRoomByID(ctx context.Context, roomID string) (*models.RoomInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	info, exists := r.rooms[roomID]
	if !exists {
		return nil, models.ErrRoomNotFound
	}
	return info, nil
}

func (r *MemoryRoomRepository) ListActiveRooms(ctx context.Context) ([]models.RoomInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]models.RoomInfo, 0, len(r.rooms))
	for _, room := range r.rooms {
		list = append(list, *room)
	}
	return list, nil
}

func (r *MemoryRoomRepository) CloseRoom(ctx context.Context, roomID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.rooms, roomID)
	return nil
}

func (r *MemoryRoomRepository) DeleteRoom(ctx context.Context, roomID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.rooms[roomID]; !exists {
		return models.ErrRoomNotFound
	}
	delete(r.rooms, roomID)
	return nil
}

type MemoryUserRepository struct {
	users map[string]*models.PeerInfo
	mu    sync.RWMutex
}

func NewMemoryUserRepository() *MemoryUserRepository {
	return &MemoryUserRepository{
		users: make(map[string]*models.PeerInfo),
	}
}

func (r *MemoryUserRepository) UpsertUser(ctx context.Context, userID, displayName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[userID] = &models.PeerInfo{
		ID:          userID,
		DisplayName: displayName,
		JoinedAt:    time.Now().UnixMilli(),
	}
	return nil
}

func (r *MemoryUserRepository) GetUserByID(ctx context.Context, userID string) (*models.PeerInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user, exists := r.users[userID]
	if !exists {
		return nil, models.ErrPeerNotFound
	}
	return user, nil
}

type MemoryParticipantRepository struct{}

func NewMemoryParticipantRepository() *MemoryParticipantRepository {
	return &MemoryParticipantRepository{}
}

func (r *MemoryParticipantRepository) AddParticipant(ctx context.Context, roomID, userID, displayName string, isHost bool) error {
	return nil
}

func (r *MemoryParticipantRepository) RemoveParticipant(ctx context.Context, roomID, userID string) error {
	return nil
}

type MemoryCallLogRepository struct{}

func NewMemoryCallLogRepository() *MemoryCallLogRepository {
	return &MemoryCallLogRepository{}
}

func (r *MemoryCallLogRepository) LogEvent(ctx context.Context, eventID, roomID, userID, eventType string, metadata interface{}) error {
	return nil
}
