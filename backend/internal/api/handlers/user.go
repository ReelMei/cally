package handlers

import (
	"net/http"

	"cally/internal/auth"
	"cally/pkg/response"
)

type UserHandler struct{}

func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	userID, displayName := auth.GetUserFromContext(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "User identity not found")
		return
	}

	response.Success(w, http.StatusOK, map[string]interface{}{
		"userId":      userID,
		"displayName": displayName,
	})
}
