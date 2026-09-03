package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"cally/internal/config"
	"cally/internal/models"
	"cally/internal/repository"
	"cally/internal/room"
)

type RoomService struct {
	manager *room.Manager
	repos   *repository.Repositories
	cfg     *config.Config
}

func NewRoomService(manager *room.Manager, repos *repository.Repositories, cfg *config.Config) *RoomService {
	return &RoomService{
		manager: manager,
		repos:   repos,
		cfg:     cfg,
	}
}

func (s *RoomService) CreateRoom(req models.CreateRoomRequest) (*models.CreateRoomResponse, error) {
	roomID := generateRoomID()
	maxPeers := req.MaxParticipants
	if maxPeers <= 0 || maxPeers > s.cfg.MaxRoomParticipants {
		maxPeers = s.cfg.MaxRoomParticipants
	}

	hostID := strings.TrimSpace(req.HostID)
	if hostID == "" {
		hostID = generateUserID()
	}

	hostDisplayName := strings.TrimSpace(req.HostDisplayName)
	if hostDisplayName == "" {
		hostDisplayName = fmt.Sprintf("Host_%s", hostID[:4])
	}

	// 1. Create in-memory active room
	r, err := s.manager.CreateRoom(roomID, req.Name, hostID, maxPeers)
	if err != nil {
		return nil, err
	}

	roomInfo := r.ToInfo()

	// 2. Persist host user and room in PostgreSQL database asynchronously / in background or inline
	if s.repos != nil && s.repos.Rooms != nil {
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_ = s.repos.Users.UpsertUser(bgCtx, hostID, hostDisplayName)
			if err := s.repos.Rooms.CreateRoom(bgCtx, &roomInfo); err != nil {
				slog.Error("failed to persist room to database", "roomId", roomID, "error", err)
			}
			_ = s.repos.CallLogs.LogEvent(bgCtx, generateEventID(), roomID, hostID, "room_created", map[string]interface{}{
				"name":     req.Name,
				"maxPeers": maxPeers,
			})
		}()
	}

	hostToken := fmt.Sprintf("token_%s_%s", roomID, hostID)

	return &models.CreateRoomResponse{
		Room:      roomInfo,
		HostToken: hostToken,
	}, nil
}

func (s *RoomService) GetRoom(roomID string) (*models.RoomInfo, error) {
	// First check real-time in-memory manager
	r, exists := s.manager.GetRoom(roomID)
	if exists {
		info := r.ToInfo()
		return &info, nil
	}

	// Fallback to PostgreSQL lookup for historical / inactive room info
	if s.repos != nil && s.repos.Rooms != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		dbRoom, err := s.repos.Rooms.GetRoomByID(ctx, roomID)
		if err == nil && dbRoom != nil {
			return dbRoom, nil
		}
	}

	return nil, models.ErrRoomNotFound
}

func (s *RoomService) ListRooms() ([]models.RoomInfo, error) {
	return s.manager.ListRooms(), nil
}

func (s *RoomService) JoinRoom(roomID string, req models.JoinRoomRequest) (*models.JoinRoomResponse, error) {
	r, exists := s.manager.GetRoom(roomID)
	if !exists {
		return nil, models.ErrRoomNotFound
	}

	if r.PeerCount() >= r.MaxPeers {
		return nil, models.ErrRoomFull
	}

	userID := strings.TrimSpace(req.UserID)
	if userID == "" {
		userID = generateUserID()
	}

	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = fmt.Sprintf("Guest_%s", userID[:4])
	}

	if _, found := r.GetPeer(userID); found {
		return nil, models.ErrPeerAlreadyJoined
	}

	isHost := (userID == r.HostID)
	token := fmt.Sprintf("token_%s_%s", roomID, userID)

	peerInfo := models.PeerInfo{
		ID:          userID,
		DisplayName: displayName,
		IsHost:      isHost,
	}

	// Log join event to PostgreSQL asynchronously
	if s.repos != nil && s.repos.Users != nil {
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_ = s.repos.Users.UpsertUser(bgCtx, userID, displayName)
			_ = s.repos.Participants.AddParticipant(bgCtx, roomID, userID, displayName, isHost)
			_ = s.repos.CallLogs.LogEvent(bgCtx, generateEventID(), roomID, userID, "peer_joined", map[string]interface{}{
				"displayName": displayName,
				"isHost":      isHost,
			})
		}()
	}

	return &models.JoinRoomResponse{
		RoomInfo: r.ToInfo(),
		Peer:     peerInfo,
		Token:    token,
	}, nil
}

func (s *RoomService) DeleteRoom(roomID, requesterID string) error {
	r, exists := s.manager.GetRoom(roomID)
	if !exists {
		return models.ErrRoomNotFound
	}

	if requesterID != "" && r.HostID != requesterID {
		return models.ErrHostOnly
	}

	err := s.manager.DeleteRoom(roomID)
	if err != nil {
		return err
	}

	// Mark closed in PostgreSQL database
	if s.repos != nil && s.repos.Rooms != nil {
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_ = s.repos.Rooms.CloseRoom(bgCtx, roomID)
			_ = s.repos.CallLogs.LogEvent(bgCtx, generateEventID(), roomID, requesterID, "room_closed", nil)
		}()
	}

	return nil
}

func (s *RoomService) GetManager() *room.Manager {
	return s.manager
}

func generateRoomID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("cally-%s", hex.EncodeToString(b))
}

func generateUserID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("user_%s", hex.EncodeToString(b))
}

func generateEventID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("evt_%s", hex.EncodeToString(b))
}
