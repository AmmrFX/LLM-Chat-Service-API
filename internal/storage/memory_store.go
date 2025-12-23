package storage

import (
	"sync"
)

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type MemoryStore struct {
	mu           sync.RWMutex
	messages     []Message
	maxExchanges int
}

// NewMemoryStore creates a new in-memory store
func NewMemoryStore(maxExchanges int) *MemoryStore {
	return &MemoryStore{
		messages:     make([]Message, 0),
		maxExchanges: maxExchanges,
	}
}

// AddMessage adds a message to the history
func (s *MemoryStore) AddMessage(msg Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = append(s.messages, msg)
	s.trimToMaxExchanges()
}

// GetMessages returns all messages (thread-safe copy)
func (s *MemoryStore) GetMessages() []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent external modifications
	result := make([]Message, len(s.messages))
	copy(result, s.messages)
	return result
}

// trimToMaxExchanges keeps only the last maxExchanges exchanges
// An exchange is a pair of user + assistant messages
func (s *MemoryStore) trimToMaxExchanges() {
	if s.maxExchanges <= 0 {
		return
	}

	// Count exchanges (pairs of user+assistant)
	exchangeCount := 0

	for i := len(s.messages) - 1; i >= 0; i-- {
		if s.messages[i].Role == "user" {
			// Check if there's an assistant message after this user message
			if i+1 < len(s.messages) && s.messages[i+1].Role == "assistant" {
				exchangeCount++
			} else if i == len(s.messages)-1 {
				// Last message is user, incomplete exchange
				break
			}
		}
	}

	// If we exceed max exchanges, trim from the beginning
	if exchangeCount > s.maxExchanges {
		// Find the start of the oldest exchange to keep
		exchangesToKeep := s.maxExchanges
		keptExchanges := 0
		startIndex := 0

		for i := 0; i < len(s.messages); i++ {
			if s.messages[i].Role == "user" {
				if i+1 < len(s.messages) && s.messages[i+1].Role == "assistant" {
					keptExchanges++
					if keptExchanges == (exchangeCount - exchangesToKeep + 1) {
						startIndex = i
						break
					}
				}
			}
		}

		s.messages = s.messages[startIndex:]
	}
}

// Clear clears all messages
func (s *MemoryStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = make([]Message, 0)
}
