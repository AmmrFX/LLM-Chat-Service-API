package error

import "errors"

var (
	ErrMessagesEmpty      = errors.New("messages cannot be empty")
	ErrInvalidRole        = errors.New("invalid role '%s' at index %d: must be 'user' or 'assistant'")
	ErrEmptyContent       = errors.New("empty content at index %d")
	ErrLastMessageNotUser = errors.New("last message must be from user, got '%s'")
	ErrGroqAPI            = errors.New("groq API error: %w")
	ErrValidation         = errors.New("validation error: %w")
	ErrStreaming          = errors.New("streaming error: %w")
	ErrCompletion         = errors.New("completion error: %w")
	ErrRedis              = errors.New("redis error: %w")
	ErrRedisCache         = errors.New("redis cache error: %w")
	ErrRedisCacheMiss     = errors.New("redis cache miss: %w")
	ErrInternal           = errors.New("internal error: %w")
	ErrNotFound           = errors.New("not found: %w")
)
