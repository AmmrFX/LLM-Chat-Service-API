package service

import (
	"time"

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

// ------------------------------------------------------------------------------------------------------
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

// ------------------------------------------------------------------------------------------------------
type ChatRequest struct {
	Messages []storage.Message `json:"messages"`
	Stream   bool              `json:"stream"`
}

// ------------------------------------------------------------------------------------------------------
func (s *chatService) ProcessChat(req *ChatRequest) (string, error) {
	if err := req.Validate(); err != nil {
		return "", err
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

// ------------------------------------------------------------------------------------------------------
func (s *chatService) ProcessChatStream(req *ChatRequest, onToken func(string) error) (string, error) {
	if err := req.Validate(); err != nil {
		return "", err
	}

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
