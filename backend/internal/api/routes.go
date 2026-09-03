package api

import (
	"net/http"
	"strings"

	"cally/internal/api/handlers"
	"cally/internal/api/middleware"
	"cally/internal/auth"
	"cally/internal/config"
	"cally/internal/room"
	"cally/internal/service"
	"cally/internal/signaling"
)

func NewRouter(cfg *config.Config, manager *room.Manager, roomService *service.RoomService) http.Handler {
	mux := http.NewServeMux()

	healthHandler := handlers.NewHealthHandler(manager, cfg.Env)
	roomHandler := handlers.NewRoomHandler(roomService)
	userHandler := handlers.NewUserHandler()
	wsHandler := signaling.NewWebSocketHandler(manager, cfg)

	// API v1 Health
	mux.Handle("/api/v1/health", healthHandler)

	// API v1 User Info
	mux.HandleFunc("/api/v1/user/me", userHandler.GetMe)

	roomsHandlerFunc := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/rooms" && r.URL.Path != "/api/v1/rooms/" {
			// Sub-path dispatcher
			routeSubPaths(w, r, roomHandler, wsHandler)
			return
		}

		switch r.Method {
		case http.MethodPost:
			roomHandler.CreateRoom(w, r)
		case http.MethodGet:
			roomHandler.ListRooms(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}

	mux.HandleFunc("/api/v1/rooms", roomsHandlerFunc)
	mux.HandleFunc("/api/v1/rooms/", roomsHandlerFunc)

	// WebSocket /ws/rooms/{roomID}
	mux.Handle("/ws/rooms/", wsHandler)

	// Global Middleware Chain: Recovery -> Logging -> CORS -> Auth -> Mux
	handler := auth.Middleware(mux)
	handler = middleware.CORS(cfg)(handler)
	handler = middleware.Logging(handler)
	handler = middleware.Recovery(handler)

	return handler
}

func routeSubPaths(w http.ResponseWriter, r *http.Request, roomHandler *handlers.RoomHandler, wsHandler *signaling.WebSocketHandler) {
	path := r.URL.Path

	if strings.HasSuffix(path, "/ws") {
		wsHandler.ServeHTTP(w, r)
		return
	}

	if strings.HasSuffix(path, "/join") {
		roomHandler.JoinRoom(w, r)
		return
	}

	if strings.HasSuffix(path, "/participants") {
		roomHandler.GetParticipants(w, r)
		return
	}

	// Default room by ID handler (GET / DELETE)
	roomHandler.HandleRoomByID(w, r)
}
