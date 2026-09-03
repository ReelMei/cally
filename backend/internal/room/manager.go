package room

import (
	"log/slog"
	"sync"
	"time"

	"cally/internal/models"
)

type Manager struct {
	rooms map[string]*Room
	mu    sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		rooms: make(map[string]*Room),
	}
}

func (m *Manager) CreateRoom(id, name, hostID string, maxPeers int) (*Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.rooms[id]; exists {
		return nil, models.ErrRoomAlreadyExists
	}

	onEmpty := func(roomID string) {
		m.onRoomEmpty(roomID)
	}

	r := NewRoom(id, name, hostID, maxPeers, onEmpty)
	m.rooms[id] = r

	go r.Run()

	slog.Info("room created", "roomId", id, "name", name, "hostId", hostID, "maxPeers", maxPeers)
	return r, nil
}

func (m *Manager) GetRoom(id string) (*Room, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	r, exists := m.rooms[id]
	return r, exists
}

func (m *Manager) DeleteRoom(id string) error {
	m.mu.Lock()
	r, exists := m.rooms[id]
	if !exists {
		m.mu.Unlock()
		return models.ErrRoomNotFound
	}

	delete(m.rooms, id)
	m.mu.Unlock()

	r.Close()
	slog.Info("room deleted", "roomId", id)
	return nil
}

func (m *Manager) RoomExists(id string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.rooms[id]
	return exists
}

func (m *Manager) ListRooms() []models.RoomInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]models.RoomInfo, 0, len(m.rooms))
	for _, r := range m.rooms {
		list = append(list, r.ToInfo())
	}
	return list
}

func (m *Manager) RoomCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.rooms)
}

func (m *Manager) onRoomEmpty(roomID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	r, exists := m.rooms[roomID]
	if !exists {
		return
	}

	if r.PeerCount() == 0 {
		delete(m.rooms, roomID)
		r.Close()
		slog.Info("empty room automatically cleaned up", "roomId", roomID)
	}
}

func (m *Manager) CleanIdleRooms(maxIdleDuration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for id, r := range m.rooms {
		if r.PeerCount() == 0 && now.Sub(r.CreatedAt) > maxIdleDuration {
			delete(m.rooms, id)
			r.Close()
			slog.Info("idle room cleaned up", "roomId", id, "age", now.Sub(r.CreatedAt))
		}
	}
}
