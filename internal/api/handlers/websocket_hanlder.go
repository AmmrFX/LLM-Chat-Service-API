package handlers

import (
	apperror "llm-chat-service/internal/error"
	"llm-chat-service/internal/service"
	"net/http"

	"go.uber.org/zap"
)

func (h *Handler) handleWebSocketChat(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("WebSocket upgrade failed", zap.Error(err))
		return
	}
	defer conn.Close()

	var req service.ChatRequest
	if err := conn.ReadJSON(&req); err != nil {
		h.logger.Error("Failed to read WebSocket message", zap.Error(err))

		errorResponse := apperror.NewErrorResponse(
			apperror.NewValidationError("Failed to read WebSocket message: invalid JSON", err),
		)

		_ = conn.WriteJSON(errorResponse)
		return
	}

	req.Stream = true

	_, err = h.chatService.ProcessChatStream(&req, func(token string) error {
		message := map[string]string{"token": token}
		return conn.WriteJSON(message)
	})

	if err != nil {
		h.logger.Error("WebSocket streaming failed", zap.Error(err))
		errorResponse := apperror.NewErrorResponse(err)
		_ = conn.WriteJSON(errorResponse)
		return
	}

	err = conn.WriteJSON(map[string]string{"done": "true"})
	if err != nil {
		h.logger.Error("Failed to write done message", zap.Error(err))
		return
	}
}
