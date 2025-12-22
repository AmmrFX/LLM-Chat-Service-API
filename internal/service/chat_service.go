package service

import (
	"fmt"
	"llm-chat-service/internal/llm"
	"llm-chat-service/internal/storage"
	"time"
)

// ChatService handles chat business logic
type ChatService struct {
	memoryStore *storage.MemoryStore
	redisStore  *storage.RedisStore
	groqClient  llm.Client
	maxTokens   int
}

// NewChatService creates a new chat service
func NewChatService(
	memoryStore *storage.MemoryStore,
	redisStore *storage.RedisStore,
	groqClient llm.Client,
	maxTokens int,
) *ChatService {
	return &ChatService{
		memoryStore: memoryStore,
		redisStore:  redisStore,
		groqClient:  groqClient,
		maxTokens:   maxTokens,
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
		return fmt.Errorf("messages cannot be empty")
	}

	// Validate each message
	for i, msg := range r.Messages {
		if msg.Role != "user" && msg.Role != "assistant" {
			return fmt.Errorf("invalid role '%s' at index %d: must be 'user' or 'assistant'", msg.Role, i)
		}
		if msg.Content == "" {
			return fmt.Errorf("empty content at index %d", i)
		}
	}

	// Last message must be from user
	lastMsg := r.Messages[len(r.Messages)-1]
	if lastMsg.Role != "user" {
		return fmt.Errorf("last message must be from user, got '%s'", lastMsg.Role)
	}

	return nil
}

// ProcessChat processes a chat request and returns the response
func (s *ChatService) ProcessChat(req *ChatRequest) (string, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return "", fmt.Errorf("validation error: %w", err)
	}

	// Get current history
	history := s.memoryStore.GetMessages()

	// Add new user message to history
	newUserMsg := req.Messages[len(req.Messages)-1]
	s.memoryStore.AddMessage(newUserMsg)

	// Prepare messages for LLM (include history + new message)
	llmMessages := append(history, newUserMsg)

	// Convert to LLM message format
	groqMessages := make([]llm.Message, len(llmMessages))
	for i, msg := range llmMessages {
		groqMessages[i] = llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Check Redis cache for token count (bonus feature)
	if s.redisStore != nil {
		cachedCount, found, err := s.redisStore.GetTokenCount(llmMessages)
		if err == nil && found {
			// Cache hit - we could optimize here, but for now just log it
			_ = cachedCount
		} else if err == nil {
			// Cache miss - compute and store
			tokenCount, err := s.redisStore.CountTokens(llmMessages)
			if err == nil {
				_ = s.redisStore.SetTokenCount(llmMessages, tokenCount, 24*time.Hour)
			}
		}
	}

	// Call Groq API
	response, err := s.groqClient.Chat(groqMessages, s.maxTokens)
	if err != nil {
		return "", fmt.Errorf("groq API error: %w", err)
	}

	// Add assistant response to history
	assistantMsg := storage.Message{
		Role:    "assistant",
		Content: response,
	}
	s.memoryStore.AddMessage(assistantMsg)

	return response, nil
}

// ProcessChatStream processes a streaming chat request
func (s *ChatService) ProcessChatStream(req *ChatRequest, onToken func(string) error) (string, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return "", fmt.Errorf("validation error: %w", err)
	}

	// Get current history
	history := s.memoryStore.GetMessages()

	// Add new user message to history
	newUserMsg := req.Messages[len(req.Messages)-1]
	s.memoryStore.AddMessage(newUserMsg)

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

	// Check Redis cache for token count
	if s.redisStore != nil {
		cachedCount, found, err := s.redisStore.GetTokenCount(llmMessages)
		if err == nil && found {
			_ = cachedCount
		} else if err == nil {
			tokenCount, err := s.redisStore.CountTokens(llmMessages)
			if err == nil {
				_ = s.redisStore.SetTokenCount(llmMessages, tokenCount, 24*time.Hour)
			}
		}
	}

	// Stream from Groq API
	response, err := s.groqClient.StreamChat(groqMessages, s.maxTokens, onToken)
	if err != nil {
		return "", fmt.Errorf("groq API error: %w", err)
	}

	// Add assistant response to history
	assistantMsg := storage.Message{
		Role:    "assistant",
		Content: response,
	}
	s.memoryStore.AddMessage(assistantMsg)

	return response, nil
}
