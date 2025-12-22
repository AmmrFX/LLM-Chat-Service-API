package config

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"llm-chat-service/internal/api"
	"llm-chat-service/internal/llm"
	"llm-chat-service/internal/logging"
	"llm-chat-service/internal/service"
	"llm-chat-service/internal/storage"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

// Config holds all configuration for the application
type Config struct {
	Port          string
	GroqAPIKey    string
	RedisAddr     string
	RedisPassword string
	MaxTokens     int
	MaxExchanges  int
	Model         string
	GroqBaseURL   string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	_ = godotenv.Load()
	cfg := &Config{
		Port:          getEnv("PORT", "8000"),
		GroqAPIKey:    getEnv("GROQ_API_KEY", ""),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		MaxTokens:     getEnvAsInt("MAX_TOKENS", 1024),
		MaxExchanges:  getEnvAsInt("MAX_EXCHANGES", 20),
		Model:         getEnv("MODEL", "llama-3.1-8b-instant"),
		GroqBaseURL:   getEnv("GROQ_BASE_URL", "https://api.groq.com/openai/v1/chat/completions"),
	}

	if cfg.GroqAPIKey == "" {
		return nil, fmt.Errorf("GROQ_API_KEY environment variable is required")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

// Factory methods for creating dependencies - all return interfaces for proper DI

// NewMessageStore creates a new message store (returns interface)
func (c *Config) NewMessageStore() storage.MessageStore {
	return storage.NewMemoryStore(c.MaxExchanges)
}

// NewCacheStore creates a new cache store (returns interface, can be nil if Redis unavailable)
func (c *Config) NewCacheStore(logger *zap.Logger) storage.CacheStore {
	redisStore, err := storage.NewRedisStore(c.RedisAddr, c.RedisPassword)
	if err != nil {
		logger.Warn("Failed to connect to Redis, continuing without cache",
			zap.Error(err),
		)
		return nil // Return nil to indicate optional dependency unavailable
	}
	logger.Info("Connected to Redis")
	return redisStore
}

// NewLLMClient creates a new LLM client (returns interface)
func (c *Config) NewLLMClient() llm.Client {
	return llm.NewGroqClient(c.GroqAPIKey, c.GroqBaseURL, c.Model)
}

// NewLogger creates and initializes the logger
func (c *Config) NewLogger() (*zap.Logger, error) {
	if err := logging.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}
	return logging.Logger, nil
}

// NewChatService creates a complete chat service with all dependencies injected via interfaces
func (c *Config) NewChatService(logger *zap.Logger) (service.ChatService, storage.CacheStore) {
	// Create message store (interface)
	messageStore := c.NewMessageStore()

	// Create cache store (interface, optional - can be nil)
	cacheStore := c.NewCacheStore(logger)

	// Create LLM client (interface)
	llmClient := c.NewLLMClient()

	// Create chat service with all interfaces injected
	chatService := service.NewChatService(messageStore, cacheStore, llmClient, c.MaxTokens)

	return chatService, cacheStore
}

// NewHandler creates a new API handler with injected dependencies
func (c *Config) NewHandler(chatService service.ChatService, logger *zap.Logger) *api.Handler {
	return api.NewHandler(chatService, logger)
}

// NewRouter creates a new router with injected dependencies
func (c *Config) NewRouter(handler *api.Handler, logger *zap.Logger) *mux.Router {
	return api.SetupRouter(handler, logger)
}

// NewHTTPServer creates a new HTTP server with injected dependencies
func (c *Config) NewHTTPServer(router *mux.Router) *http.Server {
	return &http.Server{
		Addr:         ":" + c.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}
