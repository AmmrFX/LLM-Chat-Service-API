package handlers

import (
	"encoding/json"
	"fmt"
	apperror "llm-chat-service/internal/error"
	"llm-chat-service/internal/service"
	"net/http"

	"go.uber.org/zap"
)

// ------------------------------------------------------------------------------------------------------
func (h *Handler) handleSSEChat(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req service.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", zap.Error(err))
		h.sendErrorResponse(w, apperror.NewValidationError("Invalid JSON in request body", err))
		return
	}

	req.Stream = true

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	_, err := h.chatService.ProcessChatStream(&req, func(token string) error {

		// Write SSE format: "data: token\n\n"
		data := fmt.Sprintf("data: %s\n\n", token)
		if _, err := w.Write([]byte(data)); err != nil {
			return err
		}
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		return nil
	})

	if err != nil {
		h.logger.Error("Streaming failed", zap.Error(err))

		errorResponse := apperror.NewErrorResponse(err)
		errorJSON, _ := json.Marshal(errorResponse)

		errorMsg := fmt.Sprintf("data: %s\n\n", string(errorJSON))

		_, err = w.Write([]byte(errorMsg))
		if err != nil {
			h.logger.Error("Failed to write error message", zap.Error(err))
			return
		}

		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		return
	}

	// Send completion marker
	_, err = w.Write([]byte("data: [DONE]\n\n"))
	if err != nil {
		h.logger.Error("Failed to write completion marker", zap.Error(err))
		return
	}

	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

}
