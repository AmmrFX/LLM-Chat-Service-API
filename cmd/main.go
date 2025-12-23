package main

import (
	"context"
	"fmt"
	"llm-chat-service/internal/config"
	"llm-chat-service/internal/logging"
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

	logger, err := cfg.NewLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logging.Sync()

	logger.Info("Starting LLM Chat Service",
		zap.String("port", cfg.Port),
		zap.String("redis_addr", cfg.RedisAddr),
	)

	chatService, cacheStore := cfg.NewChatService(logger)

	if cacheStore != nil {
		defer cacheStore.Close()
	}

	handler := cfg.NewHandler(chatService, logger)

	router := cfg.NewRouter(handler, logger)

	srv := cfg.NewHTTPServer(router)

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
