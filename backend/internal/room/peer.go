package room

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"cally/internal/models"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024
	sendBufferSize = 256
)

type Peer struct {
	ID            string
	DisplayName   string
	Room          *Room
	Conn          *websocket.Conn
	Send          chan []byte
	IsHost        bool
	AudioMuted    bool
	VideoOff      bool
	ScreenSharing bool
	HandRaised    bool
	JoinedAt      time.Time

	closeOnce sync.Once
	closed    bool
	mu        sync.RWMutex
}

func NewPeer(id, displayName string, isHost bool, conn *websocket.Conn, room *Room) *Peer {
	return &Peer{
		ID:            id,
		DisplayName:   displayName,
		Room:          room,
		Conn:          conn,
		Send:          make(chan []byte, sendBufferSize),
		IsHost:        isHost,
		AudioMuted:    false,
		VideoOff:      false,
		ScreenSharing: false,
		HandRaised:    false,
		JoinedAt:      time.Now(),
	}
}

func (p *Peer) ToInfo() models.PeerInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return models.PeerInfo{
		ID:            p.ID,
		DisplayName:   p.DisplayName,
		IsHost:        p.IsHost,
		AudioMuted:    p.AudioMuted,
		VideoOff:      p.VideoOff,
		ScreenSharing: p.ScreenSharing,
		HandRaised:    p.HandRaised,
		JoinedAt:      p.JoinedAt.UnixMilli(),
	}
}

func (p *Peer) ReadPump() {
	defer func() {
		if p.Room != nil {
			p.Room.Unregister <- p
		}
		if p.Conn != nil {
			_ = p.Conn.Close()
		}
	}()

	p.Conn.SetReadLimit(maxMessageSize)
	_ = p.Conn.SetReadDeadline(time.Now().Add(pongWait))
	p.Conn.SetPongHandler(func(string) error {
		_ = p.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, messageBytes, err := p.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure, websocket.CloseAbnormalClosure) {
				slog.Debug("websocket read error", "peerId", p.ID, "error", err)
			}
			break
		}

		var msg models.Message
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			slog.Warn("malformed JSON from peer", "peerId", p.ID, "error", err)
			p.SendError(400, "Invalid JSON payload")
			continue
		}

		// Security stamp: client CANNOT spoof SenderID or RoomID
		msg.SenderID = p.ID
		if p.Room != nil {
			msg.RoomID = p.Room.ID
		}
		if msg.Timestamp == 0 {
			msg.Timestamp = time.Now().UnixMilli()
		}

		if p.Room != nil {
			select {
			case p.Room.Inbound <- &msg:
			default:
				slog.Warn("room inbound channel full, dropping message", "roomId", p.Room.ID, "peerId", p.ID)
			}
		}
	}
}

func (p *Peer) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		if p.Conn != nil {
			_ = p.Conn.Close()
		}
	}()

	for {
		select {
		case message, ok := <-p.Send:
			_ = p.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Channel closed by room cleanup
				_ = p.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := p.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			// Add queued chat/signaling messages to current websocket frame if any
			n := len(p.Send)
			for i := 0; i < n; i++ {
				_, _ = w.Write([]byte{'\n'})
				_, _ = w.Write(<-p.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = p.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := p.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (p *Peer) SendJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return p.SendRaw(data)
}

func (p *Peer) SendRaw(data []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	select {
	case p.Send <- data:
		return nil
	default:
		slog.Warn("peer send buffer full, dropping message", "peerId", p.ID)
		return nil
	}
}

func (p *Peer) SendError(code int, message string) {
	errPayload, _ := json.Marshal(models.ErrorPayload{
		Code:    code,
		Message: message,
	})
	_ = p.SendJSON(models.Message{
		Type:      models.EventError,
		SenderID:  "system",
		Payload:   errPayload,
		Timestamp: time.Now().UnixMilli(),
	})
}

func (p *Peer) SafeCloseSend() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.closeOnce.Do(func() {
		p.closed = true
		close(p.Send)
	})
}

// State Mutators

func (p *Peer) SetAudioMuted(muted bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.AudioMuted = muted
}

func (p *Peer) SetVideoOff(off bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.VideoOff = off
}

func (p *Peer) SetScreenSharing(sharing bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ScreenSharing = sharing
}

func (p *Peer) SetHandRaised(raised bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.HandRaised = raised
}

func (p *Peer) SetHost(isHost bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.IsHost = isHost
}

func (p *Peer) IsHostPeer() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.IsHost
}
