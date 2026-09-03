package signaling

import (
	"cally/internal/models"
)

// Re-export event constants for convenience within signaling package
const (
	EventRoomState   = models.EventRoomState
	EventRoomEnded   = models.EventRoomEnded
	EventPeerJoined  = models.EventPeerJoined
	EventPeerLeft    = models.EventPeerLeft
	EventPeerState   = models.EventPeerState
	EventWebRTCOffer = models.EventWebRTCOffer
	EventWebRTCAnswer= models.EventWebRTCAnswer
	EventWebRTCICE   = models.EventWebRTCICE
	EventAudioState  = models.EventAudioState
	EventVideoState  = models.EventVideoState
	EventScreenState = models.EventScreenState
	EventHandRaise   = models.EventHandRaise
	EventHandLower   = models.EventHandLower
	EventChatMessage = models.EventChatMessage
	EventPeerKick    = models.EventPeerKick
	EventError       = models.EventError
)
