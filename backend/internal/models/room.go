package models

type RoomInfo struct {
	ID        string     `json:"id"`
	Name      string     `json:"name,omitempty"`
	HostID    string     `json:"hostId"`
	Peers     []PeerInfo `json:"peers"`
	MaxPeers  int        `json:"maxPeers"`
	CreatedAt int64      `json:"createdAt"`
}

type CreateRoomRequest struct {
	Name            string `json:"name"`
	MaxParticipants int    `json:"maxParticipants"`
	HostID          string `json:"hostId"`
	HostDisplayName string `json:"hostDisplayName"`
}

type CreateRoomResponse struct {
	Room      RoomInfo `json:"room"`
	HostToken string   `json:"hostToken,omitempty"`
}

type JoinRoomRequest struct {
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
}

type JoinRoomResponse struct {
	RoomInfo RoomInfo `json:"room"`
	Peer     PeerInfo `json:"peer"`
	Token    string   `json:"token,omitempty"`
}

type RoomStatePayload struct {
	RoomID string     `json:"roomId"`
	HostID string     `json:"hostId"`
	Peers  []PeerInfo `json:"peers"`
}

type MediaStatePayload struct {
	AudioMuted    *bool `json:"audioMuted,omitempty"`
	VideoOff      *bool `json:"videoOff,omitempty"`
	ScreenSharing *bool `json:"screenSharing,omitempty"`
	HandRaised    *bool `json:"handRaised,omitempty"`
}

type ChatMessagePayload struct {
	Message string `json:"message"`
	Time    int64  `json:"time,omitempty"`
}

type KickPayload struct {
	TargetID string `json:"targetId"`
	Reason   string `json:"reason,omitempty"`
}
