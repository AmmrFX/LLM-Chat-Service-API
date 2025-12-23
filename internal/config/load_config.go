package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
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

// ------------------------------------------------------------------------------------------------------
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

// ------------------------------------------------------------------------------------------------------
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ------------------------------------------------------------------------------------------------------
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
