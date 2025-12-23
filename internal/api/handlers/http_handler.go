package handlers

import (
	"encoding/json"
	apperror "llm-chat-service/internal/error"
	"llm-chat-service/internal/service"
	"net/http"

	"go.uber.org/zap"
)

// ----------------------------------------------------------------------------------------------------------------
func (h *Handler) handleJSONChat(w http.ResponseWriter, r *http.Request) {
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

	response, err := h.chatService.ProcessChat(&req)
	if err != nil {
		h.logger.Error("Chat processing failed", zap.Error(err))
		h.sendErrorResponse(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if encodeErr := json.NewEncoder(w).Encode(map[string]string{"response": response}); encodeErr != nil {
		h.logger.Error("Failed to encode response", zap.Error(encodeErr))
	}
}
