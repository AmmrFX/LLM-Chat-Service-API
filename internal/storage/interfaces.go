package storage

import "time"

// MessageStore defines the interface for storing conversation messages
type MessageStore interface {
	AddMessage(msg Message)
	GetMessages() []Message
	Clear()
}

// CacheStore defines the interface for caching operations
type CacheStore interface {
	GetTokenCount(messages []Message) (int, bool, error)
	SetTokenCount(messages []Message, count int, ttl time.Duration) error
	CountTokens(messages []Message) (int, error)
	Close() error
}
