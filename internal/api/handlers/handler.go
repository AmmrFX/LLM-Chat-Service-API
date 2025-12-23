package handlers

import (
	"encoding/json"
	"net/http"

	apperror "llm-chat-service/internal/error"
	"llm-chat-service/internal/service"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type Handler struct {
	chatService service.ChatService
	logger      *zap.Logger
	upgrader    websocket.Upgrader
}

// ------------------------------------------------------------------------------------------------------
func NewHandler(chatService service.ChatService, logger *zap.Logger) *Handler {
	return &Handler{
		chatService: chatService,
		logger:      logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

// ------------------------------------------------------------------------------------------------------
func (h *Handler) ChatHandler(w http.ResponseWriter, r *http.Request) {

	if r.Header.Get("Upgrade") == "websocket" || r.Header.Get("Connection") == "Upgrade" {
		h.handleWebSocketChat(w, r)
		return
	}

	accept := r.Header.Get("Accept")

	if accept == "text/event-stream" || r.URL.Query().Get("stream") == "true" {
		h.handleSSEChat(w, r)
		return
	}

	h.handleJSONChat(w, r)
}

// ------------------------------------------------------------------------------------------------------
func (h *Handler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	response := "OK"

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode health response", zap.Error(err))
	}
}

// ------------------------------------------------------------------------------------------------------
func (h *Handler) MetricsHandler(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("# Metrics endpoint\n"))
	if err != nil {
		h.logger.Error("Failed to write metrics endpoint", zap.Error(err))
		return
	}
}

// ------------------------------------------------------------------------------------------------------
func (h *Handler) sendErrorResponse(w http.ResponseWriter, err error) {
	statusCode := apperror.GetHTTPStatusCode(err)
	errorResponse := apperror.NewErrorResponse(err)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if encodeErr := json.NewEncoder(w).Encode(errorResponse); encodeErr != nil {
		h.logger.Error("Failed to encode error response",
			zap.Error(encodeErr),
			zap.Error(err),
		)
	}
}
