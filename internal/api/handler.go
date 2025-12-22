package api

import (
	"encoding/json"
	"fmt"
	"llm-chat-service/internal/service"
	"net/http"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Handler handles HTTP requests
type Handler struct {
	chatService service.ChatService
	logger      *zap.Logger
	upgrader    websocket.Upgrader
}

// NewHandler creates a new handler with injected dependencies
func NewHandler(chatService service.ChatService, logger *zap.Logger) *Handler {
	return &Handler{
		chatService: chatService,
		logger:      logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
	}
}

// ChatHandler handles chat requests
func (h *Handler) ChatHandler(w http.ResponseWriter, r *http.Request) {
	// Check if WebSocket upgrade is requested
	if r.Header.Get("Upgrade") == "websocket" || r.Header.Get("Connection") == "Upgrade" {
		h.handleWebSocketChat(w, r)
		return
	}

	// Check if SSE is requested
	accept := r.Header.Get("Accept")
	if accept == "text/event-stream" || r.URL.Query().Get("stream") == "true" {
		h.handleSSEChat(w, r)
		return
	}

	// Default to regular JSON response
	h.handleJSONChat(w, r)
}

// handleJSONChat handles non-streaming chat requests
func (h *Handler) handleJSONChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req service.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", zap.Error(err))
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	response, err := h.chatService.ProcessChat(&req)
	if err != nil {
		h.logger.Error("Chat processing failed", zap.Error(err))
		statusCode := http.StatusInternalServerError
		errMsg := err.Error()
		if len(errMsg) >= 14 && errMsg[:14] == "validation error" {
			statusCode = http.StatusBadRequest
		} else {
			statusCode = http.StatusBadGateway
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if encodeErr := json.NewEncoder(w).Encode(map[string]string{"error": errMsg}); encodeErr != nil {
			h.logger.Error("Failed to encode error response", zap.Error(encodeErr))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if encodeErr := json.NewEncoder(w).Encode(map[string]string{"response": response}); encodeErr != nil {
		h.logger.Error("Failed to encode response", zap.Error(encodeErr))
	}
}

// handleSSEChat handles Server-Sent Events streaming
func (h *Handler) handleSSEChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req service.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", zap.Error(err))
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	req.Stream = true

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Flush headers
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Stream tokens
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
		// Send error as SSE
		errorMsg := fmt.Sprintf("data: {\"error\": \"%s\"}\n\n", err.Error())
		_, _ = w.Write([]byte(errorMsg))
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		return
	}

	// Send completion marker
	_, _ = w.Write([]byte("data: [DONE]\n\n"))
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

// handleWebSocketChat handles WebSocket streaming
func (h *Handler) handleWebSocketChat(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("WebSocket upgrade failed", zap.Error(err))
		return
	}
	defer conn.Close()

	// Read initial message
	var req service.ChatRequest
	if err := conn.ReadJSON(&req); err != nil {
		h.logger.Error("Failed to read WebSocket message", zap.Error(err))
		return
	}

	req.Stream = true

	// Stream tokens via WebSocket
	_, err = h.chatService.ProcessChatStream(&req, func(token string) error {
		message := map[string]string{"token": token}
		return conn.WriteJSON(message)
	})

	if err != nil {
		h.logger.Error("WebSocket streaming failed", zap.Error(err))
		errorMsg := map[string]string{"error": err.Error()}
		_ = conn.WriteJSON(errorMsg)
		return
	}

	// Send completion
	_ = conn.WriteJSON(map[string]string{"done": "true"})
}

// MetricsHandler handles Prometheus metrics (bonus)
func (h *Handler) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	// This will be implemented with Prometheus client
	// For now, return a placeholder
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("# Metrics endpoint\n"))
}
