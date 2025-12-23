package service

import (
	"fmt"
	"time"

	apperror "llm-chat-service/internal/error"
	"llm-chat-service/internal/llm"
	"llm-chat-service/internal/storage"
)

// chatService handles chat business logic
type chatService struct {
	messageStore storage.MessageStore
	cacheStore   storage.CacheStore // Can be nil if caching is not available
	llmClient    llm.Client
	maxTokens    int
}

// NewChatService creates a new chat service with injected dependencies
func NewChatService(
	messageStore storage.MessageStore,
	cacheStore storage.CacheStore, // Can be nil
	llmClient llm.Client,
	maxTokens int,
) ChatService {
	return &chatService{
		messageStore: messageStore,
		cacheStore:   cacheStore,
		llmClient:    llmClient,
		maxTokens:    maxTokens,
	}
}

// ChatRequest represents the incoming chat request
type ChatRequest struct {
	Messages []storage.Message `json:"messages"`
	Stream   bool              `json:"stream"`
}

// Validate validates the chat request
func (r *ChatRequest) Validate() error {
	if len(r.Messages) == 0 {
		return apperror.NewValidationError("messages cannot be empty", nil)
	}

	// Validate each message
	for i, msg := range r.Messages {
		if msg.Role != "user" && msg.Role != "assistant" {
			return apperror.NewValidationError(
				fmt.Sprintf("invalid role '%s' at index %d: must be 'user' or 'assistant'", msg.Role, i),
				nil,
			)
		}
		if msg.Content == "" {
			return apperror.NewValidationError(
				fmt.Sprintf("empty content at index %d", i),
				nil,
			)
		}
	}

	// Last message must be from user
	lastMsg := r.Messages[len(r.Messages)-1]
	if lastMsg.Role != "user" {
		return apperror.NewValidationError(
			fmt.Sprintf("last message must be from user, got '%s'", lastMsg.Role),
			nil,
		)
	}

	return nil
}

// ProcessChat processes a chat request and returns the response
func (s *chatService) ProcessChat(req *ChatRequest) (string, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return "", err // Already wrapped with AppError
	}

	history := s.messageStore.GetMessages()

	newUserMsg := req.Messages[len(req.Messages)-1]
	s.messageStore.AddMessage(newUserMsg)

	llmMessages := append(history, newUserMsg)

	groqMessages := make([]llm.Message, len(llmMessages))
	for i, msg := range llmMessages {
		groqMessages[i] = llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Check cache for token count (bonus feature)
	if s.cacheStore != nil {
		cachedCount, found, err := s.cacheStore.GetTokenCount(llmMessages)
		if err == nil && found {
			// Cache hit - we could optimize here, but for now just log it
			_ = cachedCount
		} else if err == nil {
			// Cache miss - compute and store
			tokenCount, err := s.cacheStore.CountTokens(llmMessages)
			if err == nil {
				_ = s.cacheStore.SetTokenCount(llmMessages, tokenCount, 24*time.Hour)
			}
		}
	}

	// Call LLM API
	response, err := s.llmClient.Chat(groqMessages, s.maxTokens)
	if err != nil {
		return "", err // Already wrapped with AppError from LLM client
	}

	// Add assistant response to history
	assistantMsg := storage.Message{
		Role:    "assistant",
		Content: response,
	}
	s.messageStore.AddMessage(assistantMsg)

	return response, nil
}

// ProcessChatStream processes a streaming chat request
func (s *chatService) ProcessChatStream(req *ChatRequest, onToken func(string) error) (string, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return "", err // Already wrapped with AppError
	}

	// Get current history
	history := s.messageStore.GetMessages()

	// Add new user message to history
	newUserMsg := req.Messages[len(req.Messages)-1]
	s.messageStore.AddMessage(newUserMsg)

	// Prepare messages for LLM
	llmMessages := append(history, newUserMsg)

	// Convert to LLM message format
	groqMessages := make([]llm.Message, len(llmMessages))
	for i, msg := range llmMessages {
		groqMessages[i] = llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Check cache for token count
	if s.cacheStore != nil {
		cachedCount, found, err := s.cacheStore.GetTokenCount(llmMessages)
		if err == nil && found {
			_ = cachedCount
		} else if err == nil {
			tokenCount, err := s.cacheStore.CountTokens(llmMessages)
			if err == nil {
				_ = s.cacheStore.SetTokenCount(llmMessages, tokenCount, 24*time.Hour)
			}
		}
	}

	// Stream from LLM API
	response, err := s.llmClient.StreamChat(groqMessages, s.maxTokens, onToken)
	if err != nil {
		return "", err // Already wrapped with AppError from LLM client
	}

	// Add assistant response to history
	assistantMsg := storage.Message{
		Role:    "assistant",
		Content: response,
	}
	s.messageStore.AddMessage(assistantMsg)

	return response, nil
}
