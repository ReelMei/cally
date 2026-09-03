package models

import "encoding/json"

type Message struct {
	Type      string          `json:"type"`
	RoomID    string          `json:"roomId,omitempty"`
	SenderID  string          `json:"senderId,omitempty"`
	TargetID  string          `json:"targetId,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp int64           `json:"timestamp,omitempty"`
}

const (
	EventRoomState   = "room.state"
	EventRoomEnded   = "room.ended"
	EventPeerJoined  = "peer.joined"
	EventPeerLeft    = "peer.left"
	EventPeerState   = "peer.state"
	EventWebRTCOffer = "webrtc.offer"
	EventWebRTCAnswer= "webrtc.answer"
	EventWebRTCICE   = "webrtc.ice"
	EventAudioState  = "media.audio"
	EventVideoState  = "media.video"
	EventScreenState = "media.screen"
	EventHandRaise   = "hand.raise"
	EventHandLower   = "hand.lower"
	EventChatMessage = "chat.message"
	EventPeerKick    = "peer.kick"
	EventError       = "error"
)
