package room

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"cally/internal/models"
)

type Room struct {
	ID        string
	Name      string
	HostID    string
	MaxPeers  int
	CreatedAt time.Time

	Peers      map[string]*Peer
	Register   chan *Peer
	Unregister chan *Peer
	Inbound    chan *models.Message

	onEmpty func(roomID string)
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.RWMutex
}

func NewRoom(id, name, hostID string, maxPeers int, onEmpty func(string)) *Room {
	if maxPeers <= 0 {
		maxPeers = 6
	}
	ctx, cancel := context.WithCancel(context.Background())
	r := &Room{
		ID:         id,
		Name:       name,
		HostID:     hostID,
		MaxPeers:   maxPeers,
		CreatedAt:  time.Now(),
		Peers:      make(map[string]*Peer),
		Register:   make(chan *Peer, 32),
		Unregister: make(chan *Peer, 32),
		Inbound:    make(chan *models.Message, 256),
		onEmpty:    onEmpty,
		ctx:        ctx,
		cancel:     cancel,
	}

	return r
}

func (r *Room) Run() {
	slog.Info("room loop started", "roomId", r.ID)
	defer slog.Info("room loop ended", "roomId", r.ID)

	for {
		select {
		case <-r.ctx.Done():
			r.closeAllPeers()
			return

		case peer := <-r.Register:
			r.handleRegister(peer)

		case peer := <-r.Unregister:
			r.handleUnregister(peer)

		case msg := <-r.Inbound:
			r.handleMessage(msg)
		}
	}
}

func (r *Room) handleRegister(peer *Peer) {
	r.mu.Lock()
	currentCount := len(r.Peers)
	if currentCount >= r.MaxPeers {
		r.mu.Unlock()
		slog.Warn("room register rejected: full", "roomId", r.ID, "peerId", peer.ID)
		peer.SendError(409, "Room is full")
		peer.SafeCloseSend()
		return
	}

	// Check if sender was designated host
	if peer.ID == r.HostID || currentCount == 0 {
		peer.SetHost(true)
		r.HostID = peer.ID
	}

	r.Peers[peer.ID] = peer
	peerInfos := r.getPeerInfosLocked()
	r.mu.Unlock()

	slog.Info("peer registered", "roomId", r.ID, "peerId", peer.ID, "displayName", peer.DisplayName, "isHost", peer.IsHostPeer())

	// 1. Send room.state to new peer
	statePayload, _ := json.Marshal(models.RoomStatePayload{
		RoomID: r.ID,
		HostID: r.HostID,
		Peers:  peerInfos,
	})
	_ = peer.SendJSON(models.Message{
		Type:      models.EventRoomState,
		RoomID:    r.ID,
		SenderID:  "system",
		Payload:   statePayload,
		Timestamp: time.Now().UnixMilli(),
	})

	// 2. Notify existing participants about new peer
	joinedPayload, _ := json.Marshal(peer.ToInfo())
	r.BroadcastMessage(&models.Message{
		Type:      models.EventPeerJoined,
		RoomID:    r.ID,
		SenderID:  peer.ID,
		Payload:   joinedPayload,
		Timestamp: time.Now().UnixMilli(),
	}, peer.ID)
}

func (r *Room) handleUnregister(peer *Peer) {
	r.mu.Lock()
	existingPeer, exists := r.Peers[peer.ID]
	if !exists {
		r.mu.Unlock()
		return
	}

	delete(r.Peers, peer.ID)
	existingPeer.SafeCloseSend()

	remainingCount := len(r.Peers)
	wasHost := existingPeer.IsHostPeer()

	// If host left and peers remain, reassign host to oldest joined peer
	var newHost *Peer
	if wasHost && remainingCount > 0 {
		var oldest *Peer
		for _, p := range r.Peers {
			if oldest == nil || p.JoinedAt.Before(oldest.JoinedAt) {
				oldest = p
			}
		}
		if oldest != nil {
			oldest.SetHost(true)
			r.HostID = oldest.ID
			newHost = oldest
		}
	}
	r.mu.Unlock()

	slog.Info("peer unregistered", "roomId", r.ID, "peerId", peer.ID)

	// Notify remaining peers
	leftPayload, _ := json.Marshal(peer.ToInfo())
	r.BroadcastMessage(&models.Message{
		Type:      models.EventPeerLeft,
		RoomID:    r.ID,
		SenderID:  peer.ID,
		Payload:   leftPayload,
		Timestamp: time.Now().UnixMilli(),
	}, "")

	if newHost != nil {
		// Broadcast host reassignment event via room.state update
		r.broadcastRoomState()
	}

	if remainingCount == 0 && r.onEmpty != nil {
		slog.Info("room empty, triggering cleanup", "roomId", r.ID)
		go r.onEmpty(r.ID)
	}
}

func (r *Room) handleMessage(msg *models.Message) {
	r.mu.RLock()
	sender, senderExists := r.Peers[msg.SenderID]
	r.mu.RUnlock()

	if !senderExists {
		slog.Warn("message received from unknown peer", "roomId", r.ID, "senderId", msg.SenderID)
		return
	}

	switch msg.Type {

	// WebRTC Signaling (Offer, Answer, ICE) - Targeted routing
	case models.EventWebRTCOffer, models.EventWebRTCAnswer, models.EventWebRTCICE:
		if msg.TargetID == "" {
			sender.SendError(400, fmt.Sprintf("Target peer ID required for %s", msg.Type))
			return
		}
		if msg.TargetID == sender.ID {
			sender.SendError(400, "Target peer cannot be sender")
			return
		}

		r.mu.RLock()
		targetPeer, targetExists := r.Peers[msg.TargetID]
		r.mu.RUnlock()

		if !targetExists {
			sender.SendError(404, fmt.Sprintf("Target peer %s not found in room", msg.TargetID))
			return
		}

		_ = targetPeer.SendJSON(msg)

	// Media State updates
	case models.EventAudioState:
		var payload models.MediaStatePayload
		if err := json.Unmarshal(msg.Payload, &payload); err == nil && payload.AudioMuted != nil {
			sender.SetAudioMuted(*payload.AudioMuted)
			r.broadcastPeerState(sender)
		}

	case models.EventVideoState:
		var payload models.MediaStatePayload
		if err := json.Unmarshal(msg.Payload, &payload); err == nil && payload.VideoOff != nil {
			sender.SetVideoOff(*payload.VideoOff)
			r.broadcastPeerState(sender)
		}

	case models.EventScreenState:
		var payload models.MediaStatePayload
		if err := json.Unmarshal(msg.Payload, &payload); err == nil && payload.ScreenSharing != nil {
			sender.SetScreenSharing(*payload.ScreenSharing)
			r.broadcastPeerState(sender)
		}

	case models.EventHandRaise:
		sender.SetHandRaised(true)
		r.broadcastPeerState(sender)

	case models.EventHandLower:
		sender.SetHandRaised(false)
		r.broadcastPeerState(sender)

	case models.EventChatMessage:
		if msg.TargetID != "" {
			r.mu.RLock()
			targetPeer, targetExists := r.Peers[msg.TargetID]
			r.mu.RUnlock()
			if targetExists {
				_ = targetPeer.SendJSON(msg)
			}
		} else {
			r.BroadcastMessage(msg, "")
		}

	// Host Operations
	case models.EventPeerKick:
		if !sender.IsHostPeer() {
			sender.SendError(403, "Only the host can kick participants")
			return
		}
		var kickPayload models.KickPayload
		if err := json.Unmarshal(msg.Payload, &kickPayload); err != nil || kickPayload.TargetID == "" {
			sender.SendError(400, "Invalid kick payload: targetId required")
			return
		}

		r.mu.RLock()
		targetPeer, targetExists := r.Peers[kickPayload.TargetID]
		r.mu.RUnlock()

		if targetExists {
			slog.Info("host kicking peer", "roomId", r.ID, "hostId", sender.ID, "targetId", targetPeer.ID)
			targetPeer.SendError(403, "You have been kicked from the room by the host")
			go func(p *Peer) {
				time.Sleep(100 * time.Millisecond)
				r.Unregister <- p
			}(targetPeer)
		} else {
			sender.SendError(404, "Target peer not found")
		}

	default:
		slog.Warn("unhandled signaling message type", "roomId", r.ID, "type", msg.Type)
		sender.SendError(400, fmt.Sprintf("Unsupported message type: %s", msg.Type))
	}
}

func (r *Room) broadcastPeerState(peer *Peer) {
	peerInfoPayload, _ := json.Marshal(peer.ToInfo())
	r.BroadcastMessage(&models.Message{
		Type:      models.EventPeerState,
		RoomID:    r.ID,
		SenderID:  peer.ID,
		Payload:   peerInfoPayload,
		Timestamp: time.Now().UnixMilli(),
	}, "")
}

func (r *Room) broadcastRoomState() {
	statePayload, _ := json.Marshal(models.RoomStatePayload{
		RoomID: r.ID,
		HostID: r.HostID,
		Peers:  r.ListPeers(),
	})
	r.BroadcastMessage(&models.Message{
		Type:      models.EventRoomState,
		RoomID:    r.ID,
		SenderID:  "system",
		Payload:   statePayload,
		Timestamp: time.Now().UnixMilli(),
	}, "")
}

func (r *Room) BroadcastMessage(msg *models.Message, excludeID string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, peer := range r.Peers {
		if excludeID != "" && peer.ID == excludeID {
			continue
		}
		_ = peer.SendJSON(msg)
	}
}

func (r *Room) Close() {
	r.cancel()
}

func (r *Room) closeAllPeers() {
	r.mu.Lock()
	defer r.mu.Unlock()

	endedMsg, _ := json.Marshal(models.Message{
		Type:      models.EventRoomEnded,
		RoomID:    r.ID,
		SenderID:  "system",
		Timestamp: time.Now().UnixMilli(),
	})

	for _, peer := range r.Peers {
		_ = peer.SendRaw(endedMsg)
		peer.SafeCloseSend()
	}
	r.Peers = make(map[string]*Peer)
}

func (r *Room) ToInfo() models.RoomInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return models.RoomInfo{
		ID:        r.ID,
		Name:      r.Name,
		HostID:    r.HostID,
		Peers:     r.getPeerInfosLocked(),
		MaxPeers:  r.MaxPeers,
		CreatedAt: r.CreatedAt.UnixMilli(),
	}
}

func (r *Room) getPeerInfosLocked() []models.PeerInfo {
	infos := make([]models.PeerInfo, 0, len(r.Peers))
	for _, p := range r.Peers {
		infos = append(infos, p.ToInfo())
	}
	return infos
}

func (r *Room) GetPeer(id string) (*Peer, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.Peers[id]
	return p, ok
}

func (r *Room) ListPeers() []models.PeerInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.getPeerInfosLocked()
}

func (r *Room) PeerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.Peers)
}
