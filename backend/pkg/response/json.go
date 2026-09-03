package response

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func JSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func Success(w http.ResponseWriter, statusCode int, data interface{}) {
	if data == nil {
		data = map[string]interface{}{}
	}
	JSON(w, statusCode, Response{
		Success: true,
		Data:    data,
	})
}

func Error(w http.ResponseWriter, statusCode int, code string, message string) {
	JSON(w, statusCode, Response{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
	})
}
