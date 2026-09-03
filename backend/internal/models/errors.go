package models

import "errors"

var (
	ErrRoomNotFound      = errors.New("room not found")
	ErrRoomFull          = errors.New("room limit reached")
	ErrRoomAlreadyExists = errors.New("room with this ID already exists")
	ErrPeerNotFound      = errors.New("peer not found")
	ErrPeerAlreadyJoined = errors.New("peer with this ID is already in the room")
	ErrUnauthorized      = errors.New("unauthorized action")
	ErrHostOnly          = errors.New("only the room host can perform this action")
	ErrInvalidPayload    = errors.New("invalid or malformed request payload")
	ErrTargetNotFound    = errors.New("target peer not found in this room")
	ErrSamePeerTarget    = errors.New("target peer cannot be the sender")
)

type ErrorPayload struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
