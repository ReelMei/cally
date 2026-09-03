package handlers

import (
	"net/http"
	"time"

	"cally/internal/room"
	"cally/pkg/response"
)

type HealthHandler struct {
	manager   *room.Manager
	startTime time.Time
	env       string
}

func NewHealthHandler(manager *room.Manager, env string) *HealthHandler {
	return &HealthHandler{
		manager:   manager,
		startTime: time.Now(),
		env:       env,
	}
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only GET method is allowed")
		return
	}

	uptime := time.Since(h.startTime).String()

	response.Success(w, http.StatusOK, map[string]interface{}{
		"status":      "ok",
		"environment": h.env,
		"uptime":       uptime,
		"activeRooms": h.manager.RoomCount(),
		"timestamp":   time.Now().UnixMilli(),
	})
}
