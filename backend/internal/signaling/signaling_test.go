package signaling

import (
	"testing"

	"cally/internal/models"
)

func TestValidateSignalingMessage(t *testing.T) {
	// Missing type
	msg := &models.Message{}
	valid, reason := ValidateSignalingMessage(msg)
	if valid {
		t.Errorf("expected message without type to be invalid")
	}
	if reason != "Message type is required" {
		t.Errorf("unexpected error reason: %s", reason)
	}

	// WebRTC Offer missing target ID
	msg = &models.Message{
		Type: EventWebRTCOffer,
	}
	valid, reason = ValidateSignalingMessage(msg)
	if valid {
		t.Errorf("expected WebRTC offer without targetId to be invalid")
	}

	// Valid WebRTC Offer
	msg = &models.Message{
		Type:     EventWebRTCOffer,
		SenderID: "user-1",
		TargetID: "user-2",
	}
	valid, _ = ValidateSignalingMessage(msg)
	if !valid {
		t.Errorf("expected WebRTC offer with targetId to be valid")
	}
}
