package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"cally/internal/config"
	"cally/internal/models"
	"cally/internal/repository"
	"cally/internal/room"
	"cally/internal/service"
)

func TestHealthEndpoint(t *testing.T) {
	mgr := room.NewManager()
	handler := NewHealthHandler(mgr, "test")

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", w.Code)
	}

	var res map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}

	if res["success"] != true {
		t.Errorf("expected success true, got %v", res["success"])
	}
}

func TestCreateAndGetRoomAPI(t *testing.T) {
	cfg, _ := config.Load()
	mgr := room.NewManager()
	repos := &repository.Repositories{
		Rooms:        repository.NewMemoryRoomRepository(),
		Users:        repository.NewMemoryUserRepository(),
		Participants: repository.NewMemoryParticipantRepository(),
		CallLogs:     repository.NewMemoryCallLogRepository(),
	}
	svc := service.NewRoomService(mgr, repos, cfg)
	handler := NewRoomHandler(svc)

	// 1. Create Room
	createBody := models.CreateRoomRequest{
		Name:            "API Test Room",
		MaxParticipants: 4,
		HostDisplayName: "Tester",
	}
	bodyBytes, _ := json.Marshal(createBody)

	req := httptest.NewRequest("POST", "/api/v1/rooms", bytes.NewBuffer(bodyBytes))
	w := httptest.NewRecorder()

	handler.CreateRoom(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d. Body: %s", w.Code, w.Body.String())
	}

	var createRes map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&createRes)
	data := createRes["data"].(map[string]interface{})
	roomData := data["room"].(map[string]interface{})
	roomID := roomData["id"].(string)

	if roomID == "" {
		t.Fatalf("expected generated room ID")
	}

	// 2. Fetch Room
	getReq := httptest.NewRequest("GET", "/api/v1/rooms/"+roomID, nil)
	getW := httptest.NewRecorder()

	handler.HandleRoomByID(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK for GET room, got %d", getW.Code)
	}
}
