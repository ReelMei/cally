package signaling

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"cally/internal/auth"
	"cally/internal/config"
	"cally/internal/room"
	"cally/pkg/response"

	"github.com/gorilla/websocket"
)

type WebSocketHandler struct {
	manager  *room.Manager
	cfg      *config.Config
	upgrader websocket.Upgrader
}

func NewWebSocketHandler(manager *room.Manager, cfg *config.Config) *WebSocketHandler {
	return &WebSocketHandler{
		manager: manager,
		cfg:     cfg,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				return cfg.IsOriginAllowed(origin)
			},
		},
	}
}

func (h *WebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract roomID from path
	roomID := extractRoomIDFromPath(r.URL.Path)
	if roomID == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_ROOM_ID", "Room ID is required")
		return
	}

	// 1. Validate room exists
	rm, exists := h.manager.GetRoom(roomID)
	if !exists {
		slog.Warn("ws upgrade rejected: room not found", "roomId", roomID, "remote", r.RemoteAddr)
		response.Error(w, http.StatusNotFound, "ROOM_NOT_FOUND", "Room does not exist")
		return
	}

	// 2. Validate participant identity
	userID, displayName := auth.GetUserFromContext(r.Context())
	if userID == "" {
		userID = r.URL.Query().Get("userId")
	}
	if displayName == "" {
		displayName = r.URL.Query().Get("displayName")
	}

	if userID == "" {
		userID = generateRandomID("peer")
	}
	if displayName == "" {
		displayName = fmt.Sprintf("User_%s", userID[len(userID)-4:])
	}

	isHost := (userID == rm.HostID)

	// 3. Upgrade HTTP connection
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "error", err, "roomId", roomID)
		return
	}

	// 4. Create Peer
	peer := room.NewPeer(userID, displayName, isHost, conn, rm)

	slog.Info("websocket connection established",
		"roomId", roomID,
		"peerId", userID,
		"displayName", displayName,
		"isHost", isHost,
		"remote", r.RemoteAddr,
	)

	// 5. Register Peer with Room
	rm.Register <- peer

	// 6. Start WritePump and ReadPump goroutines
	go peer.WritePump()
	peer.ReadPump() // Synchronous read pump blocks until disconnect
}

func extractRoomIDFromPath(path string) string {
	// Handles both /ws/rooms/{roomID} and /api/v1/rooms/{roomID}/ws
	if strings.HasPrefix(path, "/ws/rooms/") {
		trimmed := strings.TrimPrefix(path, "/ws/rooms/")
		return strings.Split(trimmed, "/")[0]
	}
	if strings.HasPrefix(path, "/api/v1/rooms/") {
		trimmed := strings.TrimPrefix(path, "/api/v1/rooms/")
		trimmed = strings.TrimSuffix(trimmed, "/ws")
		return strings.Split(trimmed, "/")[0]
	}
	return ""
}

func generateRandomID(prefix string) string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(b))
}
