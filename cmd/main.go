package main

import (
	"context"
	"fmt"
	"llm-chat-service/internal/api"
	"llm-chat-service/internal/config"
	"llm-chat-service/internal/llm"
	"llm-chat-service/internal/logging"
	"llm-chat-service/internal/service"
	"llm-chat-service/internal/storage"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := logging.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logging.Sync()

	logger := logging.Logger
	logger.Info("Starting LLM Chat Service",
		zap.String("port", cfg.Port),
		zap.String("redis_addr", cfg.RedisAddr),
	)

	// Initialize storage
	memoryStore := storage.NewMemoryStore(cfg.MaxExchanges)

	// Initialize Redis (optional, continue if it fails)
	var redisStore *storage.RedisStore
	redisStore, err = storage.NewRedisStore(cfg.RedisAddr, cfg.RedisPassword)
	if err != nil {
		logger.Warn("Failed to connect to Redis, continuing without cache",
			zap.Error(err),
		)
		redisStore = nil
	} else {
		defer redisStore.Close()
		logger.Info("Connected to Redis")
	}

	// Initialize Groq client
	groqClient := llm.NewGroqClient(cfg.GroqAPIKey)

	// Initialize chat service
	chatService := service.NewChatService(memoryStore, redisStore, groqClient, cfg.MaxTokens)

	// Initialize handler
	handler := api.NewHandler(chatService, logger)

	// Setup router
	router := api.SetupRouter(handler, logger)

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Server starting", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server stopped")
}
