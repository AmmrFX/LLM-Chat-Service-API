package config

import (
	"fmt"
	"llm-chat-service/internal/api"
	"llm-chat-service/internal/api/handlers"
	"llm-chat-service/internal/llm"
	"llm-chat-service/internal/logging"
	"llm-chat-service/internal/service"
	"llm-chat-service/internal/storage"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// ------------------------------------------------------------------------------------------------------
func (c *Config) NewMessageStore() storage.MessageStore {
	return storage.NewMemoryStore(c.MaxExchanges)
}

// ------------------------------------------------------------------------------------------------------
func (c *Config) NewCacheStore(logger *zap.Logger) storage.CacheStore {
	redisStore, err := storage.NewRedisStore(c.RedisAddr, c.RedisPassword)
	if err != nil {
		logger.Warn("Failed to connect to Redis, continuing without cache",
			zap.Error(err),
		)
		return nil
	}
	logger.Info("Connected to Redis")
	return redisStore
}

// ------------------------------------------------------------------------------------------------------
func (c *Config) NewLLMClient() llm.Client {
	return llm.NewGroqClient(c.GroqAPIKey, c.GroqBaseURL, c.Model)
}

// ------------------------------------------------------------------------------------------------------
func (c *Config) NewLogger() (*zap.Logger, error) {
	if err := logging.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}
	return logging.Logger, nil
}

// ------------------------------------------------------------------------------------------------------
func (c *Config) NewChatService(logger *zap.Logger) (service.ChatService, storage.CacheStore) {
	// Create message store
	messageStore := c.NewMessageStore()

	// Create cache store
	cacheStore := c.NewCacheStore(logger)

	// Create LLM client
	llmClient := c.NewLLMClient()

	chatService := service.NewChatService(messageStore, cacheStore, llmClient, c.MaxTokens)

	return chatService, cacheStore
}

// ------------------------------------------------------------------------------------------------------
func (c *Config) NewHandler(chatService service.ChatService, logger *zap.Logger) *handlers.Handler {
	return handlers.NewHandler(chatService, logger)
}

// ------------------------------------------------------------------------------------------------------
func (c *Config) NewRouter(handler *handlers.Handler, logger *zap.Logger) *mux.Router {
	return api.SetupRouter(handler, logger)
}

// ------------------------------------------------------------------------------------------------------
func (c *Config) NewHTTPServer(router *mux.Router) *http.Server {
	return &http.Server{
		Addr:         ":" + c.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}
