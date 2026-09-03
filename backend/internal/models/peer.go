package models

type PeerInfo struct {
	ID            string `json:"id"`
	DisplayName   string `json:"displayName"`
	IsHost        bool   `json:"isHost"`
	AudioMuted    bool   `json:"audioMuted"`
	VideoOff      bool   `json:"videoOff"`
	ScreenSharing bool   `json:"screenSharing"`
	HandRaised    bool   `json:"handRaised"`
	JoinedAt      int64  `json:"joinedAt"`
}
