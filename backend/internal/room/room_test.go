package room

import (
	"encoding/json"
	"testing"
	"time"

	"cally/internal/models"
)

func TestPeerStateMutations(t *testing.T) {
	p := NewPeer("peer-1", "Alice", true, nil, nil)

	info := p.ToInfo()
	if info.ID != "peer-1" || info.DisplayName != "Alice" || !info.IsHost {
		t.Errorf("unexpected initial peer info: %+v", info)
	}

	p.SetAudioMuted(true)
	p.SetVideoOff(true)
	p.SetScreenSharing(true)
	p.SetHandRaised(true)

	info = p.ToInfo()
	if !info.AudioMuted || !info.VideoOff || !info.ScreenSharing || !info.HandRaised {
		t.Errorf("state mutation failed: %+v", info)
	}
}

func TestRoomPeerRegistrationAndCapacity(t *testing.T) {
	r := NewRoom("room-cap", "Capacity Test", "host-1", 2, nil)
	go r.Run()
	defer r.Close()

	peer1 := NewPeer("host-1", "Host Peer", true, nil, r)
	peer2 := NewPeer("user-2", "User 2", false, nil, r)

	r.Register <- peer1
	r.Register <- peer2

	// Allow goroutine event loop to process
	time.Sleep(50 * time.Millisecond)

	if r.PeerCount() != 2 {
		t.Errorf("expected 2 peers, got %d", r.PeerCount())
	}

	// Attempt to register 3rd peer when max is 2
	peer3 := NewPeer("user-3", "User 3", false, nil, r)
	r.Register <- peer3

	time.Sleep(50 * time.Millisecond)

	if r.PeerCount() != 2 {
		t.Errorf("expected room count to remain 2, got %d", r.PeerCount())
	}
}

func TestRoomMessageHandlingHostKick(t *testing.T) {
	r := NewRoom("room-kick", "Kick Test", "host-1", 4, nil)
	go r.Run()
	defer r.Close()

	hostPeer := NewPeer("host-1", "Host", true, nil, r)
	userPeer := NewPeer("user-2", "User 2", false, nil, r)

	r.Register <- hostPeer
	r.Register <- userPeer

	time.Sleep(50 * time.Millisecond)

	// Non-host attempts to kick host (should fail)
	kickPayload, _ := json.Marshal(models.KickPayload{
		TargetID: "host-1",
	})
	r.Inbound <- &models.Message{
		Type:     models.EventPeerKick,
		SenderID: "user-2",
		RoomID:   r.ID,
		Payload:  kickPayload,
	}

	time.Sleep(50 * time.Millisecond)

	if r.PeerCount() != 2 {
		t.Errorf("expected non-host kick to be rejected, but peer count changed to %d", r.PeerCount())
	}
}
