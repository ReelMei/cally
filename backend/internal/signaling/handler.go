package signaling

import (
	"cally/internal/models"
)

// ValidateSignalingMessage checks envelope fields for mandatory values
func ValidateSignalingMessage(msg *models.Message) (bool, string) {
	if msg.Type == "" {
		return false, "Message type is required"
	}

	switch msg.Type {
	case EventWebRTCOffer, EventWebRTCAnswer, EventWebRTCICE:
		if msg.TargetID == "" {
			return false, "Target peer ID required for WebRTC signaling"
		}
	}

	return true, ""
}
