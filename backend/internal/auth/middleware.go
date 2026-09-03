package auth

import (
	"context"
	"net/http"
	"strings"

	"cally/pkg/response"
)

type contextKey string

const (
	UserIDKey      contextKey = "userId"
	DisplayNameKey contextKey = "displayName"
	TokenKey       contextKey = "authToken"
)

type UserContext struct {
	UserID      string
	DisplayName string
	Token       string
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := ""
		displayName := ""
		token := ""

		// 1. Check Authorization header
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
			// Format: token_<roomId>_<userId> or similar
			parts := strings.Split(token, "_")
			if len(parts) >= 3 {
				userID = parts[len(parts)-1]
			}
		}

		// 2. Check query params (common for WebSocket connections)
		if userID == "" {
			userID = r.URL.Query().Get("userId")
		}
		if displayName == "" {
			displayName = r.URL.Query().Get("displayName")
		}
		if token == "" {
			token = r.URL.Query().Get("token")
		}

		// Inject into context
		ctx := r.Context()
		if userID != "" {
			ctx = context.WithValue(ctx, UserIDKey, userID)
		}
		if displayName != "" {
			ctx = context.WithValue(ctx, DisplayNameKey, displayName)
		}
		if token != "" {
			ctx = context.WithValue(ctx, TokenKey, token)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value(UserIDKey).(string)
		if !ok || userID == "" {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
			return
		}
		next(w, r)
	}
}

func GetUserFromContext(ctx context.Context) (UserID, DisplayName string) {
	if val, ok := ctx.Value(UserIDKey).(string); ok {
		UserID = val
	}
	if val, ok := ctx.Value(DisplayNameKey).(string); ok {
		DisplayName = val
	}
	return UserID, DisplayName
}
