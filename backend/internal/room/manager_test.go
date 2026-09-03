package room

import (
	"fmt"
	"sync"
	"testing"

	"cally/internal/models"
)

func TestManagerCreateAndGetRoom(t *testing.T) {
	mgr := NewManager()

	room, err := mgr.CreateRoom("room-1", "Test Room", "host-1", 6)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if room.ID != "room-1" {
		t.Errorf("expected room ID 'room-1', got %s", room.ID)
	}
	if room.Name != "Test Room" {
		t.Errorf("expected room name 'Test Room', got %s", room.Name)
	}

	fetched, exists := mgr.GetRoom("room-1")
	if !exists {
		t.Fatalf("expected room to exist")
	}
	if fetched.ID != "room-1" {
		t.Errorf("expected fetched room ID 'room-1', got %s", fetched.ID)
	}

	// Test duplicate creation error
	_, err = mgr.CreateRoom("room-1", "Duplicate Room", "host-2", 6)
	if err != models.ErrRoomAlreadyExists {
		t.Errorf("expected ErrRoomAlreadyExists, got %v", err)
	}
}

func TestManagerDeleteRoom(t *testing.T) {
	mgr := NewManager()

	_, _ = mgr.CreateRoom("room-to-delete", "Delete Me", "host-1", 4)

	if !mgr.RoomExists("room-to-delete") {
		t.Fatalf("expected room to exist before deletion")
	}

	err := mgr.DeleteRoom("room-to-delete")
	if err != nil {
		t.Fatalf("expected no error on delete, got %v", err)
	}

	if mgr.RoomExists("room-to-delete") {
		t.Errorf("expected room to be deleted")
	}

	// Delete non-existent room
	err = mgr.DeleteRoom("non-existent")
	if err != models.ErrRoomNotFound {
		t.Errorf("expected ErrRoomNotFound, got %v", err)
	}
}

func TestManagerConcurrentAccess(t *testing.T) {
	mgr := NewManager()
	var wg sync.WaitGroup

	numGoroutines := 50
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			roomID := fmt.Sprintf("room-%d", id)
			_, _ = mgr.CreateRoom(roomID, fmt.Sprintf("Room %d", id), fmt.Sprintf("host-%d", id), 4)
			_, _ = mgr.GetRoom(roomID)
			_ = mgr.ListRooms()
		}(i)
	}

	wg.Wait()

	if mgr.RoomCount() != numGoroutines {
		t.Errorf("expected %d rooms, got %d", numGoroutines, mgr.RoomCount())
	}
}
