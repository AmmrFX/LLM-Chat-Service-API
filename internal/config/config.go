package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all configuration for the application
type Config struct {
	Port          string
	GroqAPIKey    string
	RedisAddr     string
	RedisPassword string
	MaxTokens     int
	MaxExchanges  int
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Port:          getEnv("PORT", "8000"),
		GroqAPIKey:    getEnv("GROQ_API_KEY", ""),
		RedisAddr:     getEnv("REDIS_ADDR", "redis:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		MaxTokens:     getEnvAsInt("MAX_TOKENS", 1024),
		MaxExchanges:  getEnvAsInt("MAX_EXCHANGES", 20),
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
