package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"cally/internal/auth"
	"cally/internal/models"
	"cally/internal/service"
	"cally/pkg/response"
)

type RoomHandler struct {
	roomService *service.RoomService
}

func NewRoomHandler(roomService *service.RoomService) *RoomHandler {
	return &RoomHandler{
		roomService: roomService,
	}
}

// CreateRoom handles POST /api/v1/rooms
func (h *RoomHandler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	var req models.CreateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && r.ContentLength > 0 {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Malformed JSON body")
		return
	}

	// Override HostID if present in Context
	ctxUserID, _ := auth.GetUserFromContext(r.Context())
	if ctxUserID != "" {
		req.HostID = ctxUserID
	}

	res, err := h.roomService.CreateRoom(req)
	if err != nil {
		if errors.Is(err, models.ErrRoomAlreadyExists) {
			response.Error(w, http.StatusConflict, "ROOM_EXISTS", err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, "CREATE_ROOM_FAILED", err.Error())
		return
	}

	response.Success(w, http.StatusCreated, res)
}

// ListRooms handles GET /api/v1/rooms
func (h *RoomHandler) ListRooms(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	rooms, err := h.roomService.ListRooms()
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "LIST_ROOMS_FAILED", err.Error())
		return
	}

	response.Success(w, http.StatusOK, map[string]interface{}{
		"rooms": rooms,
		"count": len(rooms),
	})
}

// HandleRoomByID routes GET /api/v1/rooms/{roomID}, DELETE /api/v1/rooms/{roomID}
func (h *RoomHandler) HandleRoomByID(w http.ResponseWriter, r *http.Request) {
	roomID := extractRoomID(r.URL.Path, "/api/v1/rooms/")
	if roomID == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_ROOM_ID", "Room ID is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		roomInfo, err := h.roomService.GetRoom(roomID)
		if err != nil {
			if errors.Is(err, models.ErrRoomNotFound) {
				response.Error(w, http.StatusNotFound, "ROOM_NOT_FOUND", "Room does not exist")
				return
			}
			response.Error(w, http.StatusInternalServerError, "GET_ROOM_FAILED", err.Error())
			return
		}
		response.Success(w, http.StatusOK, roomInfo)

	case http.MethodDelete:
		requesterID, _ := auth.GetUserFromContext(r.Context())
		err := h.roomService.DeleteRoom(roomID, requesterID)
		if err != nil {
			if errors.Is(err, models.ErrRoomNotFound) {
				response.Error(w, http.StatusNotFound, "ROOM_NOT_FOUND", "Room does not exist")
				return
			}
			if errors.Is(err, models.ErrHostOnly) {
				response.Error(w, http.StatusForbidden, "HOST_ONLY", "Only the room host can delete the room")
				return
			}
			response.Error(w, http.StatusInternalServerError, "DELETE_ROOM_FAILED", err.Error())
			return
		}
		response.Success(w, http.StatusOK, map[string]interface{}{
			"message": "Room deleted successfully",
			"roomId":  roomID,
		})

	default:
		response.Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}
}

// JoinRoom handles POST /api/v1/rooms/{roomID}/join
func (h *RoomHandler) JoinRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	roomID := extractSubPath(r.URL.Path, "/api/v1/rooms/", "/join")
	if roomID == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_ROOM_ID", "Room ID is required")
		return
	}

	var req models.JoinRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && r.ContentLength > 0 {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Malformed JSON payload")
		return
	}

	ctxUserID, ctxDisplayName := auth.GetUserFromContext(r.Context())
	if req.UserID == "" && ctxUserID != "" {
		req.UserID = ctxUserID
	}
	if req.DisplayName == "" && ctxDisplayName != "" {
		req.DisplayName = ctxDisplayName
	}

	res, err := h.roomService.JoinRoom(roomID, req)
	if err != nil {
		if errors.Is(err, models.ErrRoomNotFound) {
			response.Error(w, http.StatusNotFound, "ROOM_NOT_FOUND", "Room does not exist")
			return
		}
		if errors.Is(err, models.ErrRoomFull) {
			response.Error(w, http.StatusConflict, "ROOM_FULL", "Room limit reached")
			return
		}
		if errors.Is(err, models.ErrPeerAlreadyJoined) {
			response.Error(w, http.StatusConflict, "ALREADY_JOINED", "Participant is already in room")
			return
		}
		response.Error(w, http.StatusInternalServerError, "JOIN_ROOM_FAILED", err.Error())
		return
	}

	response.Success(w, http.StatusOK, res)
}

// GetParticipants handles GET /api/v1/rooms/{roomID}/participants
func (h *RoomHandler) GetParticipants(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	roomID := extractSubPath(r.URL.Path, "/api/v1/rooms/", "/participants")
	if roomID == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_ROOM_ID", "Room ID is required")
		return
	}

	roomInfo, err := h.roomService.GetRoom(roomID)
	if err != nil {
		if errors.Is(err, models.ErrRoomNotFound) {
			response.Error(w, http.StatusNotFound, "ROOM_NOT_FOUND", "Room does not exist")
			return
		}
		response.Error(w, http.StatusInternalServerError, "GET_PARTICIPANTS_FAILED", err.Error())
		return
	}

	response.Success(w, http.StatusOK, map[string]interface{}{
		"roomId":       roomID,
		"participants": roomInfo.Peers,
		"count":        len(roomInfo.Peers),
	})
}

// Helper functions for path parsing
func extractRoomID(path, prefix string) string {
	trimmed := strings.TrimPrefix(path, prefix)
	parts := strings.Split(trimmed, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func extractSubPath(path, prefix, suffix string) string {
	trimmed := strings.TrimPrefix(path, prefix)
	trimmed = strings.TrimSuffix(trimmed, suffix)
	parts := strings.Split(trimmed, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}
