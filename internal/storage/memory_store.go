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

// ------------------------------------------------------------------------------------------------------
// NewMemoryStore creates a new in-memory store
func NewMemoryStore(maxExchanges int) *MemoryStore {
	return &MemoryStore{
		messages:     make([]Message, 0),
		maxExchanges: maxExchanges,
	}
}

// ------------------------------------------------------------------------------------------------------
func (s *MemoryStore) AddMessage(msg Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = append(s.messages, msg)
	s.trimToMaxExchanges()
}

// ------------------------------------------------------------------------------------------------------
func (s *MemoryStore) GetMessages() []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Message, len(s.messages))
	copy(result, s.messages)
	return result
}

// ------------------------------------------------------------------------------------------------------
// An exchange is a pair of user + assistant messages
func (s *MemoryStore) trimToMaxExchanges() {
	if s.maxExchanges <= 0 {
		return
	}

	exchangeCount := countExchanges(s.messages)

	if exchangeCount > s.maxExchanges {
		exchangesToKeep := s.maxExchanges
		startIndex := findStartIndex(s.messages, exchangeCount, exchangesToKeep)
		s.messages = s.messages[startIndex:]
	}
}

// ------------------------------------------------------------------------------------------------------
// countExchanges counts the number of complete exchanges (user + assistant pairs) in messages
func countExchanges(messages []Message) int {
	exchangeCount := 0

	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			if i+1 < len(messages) && messages[i+1].Role == "assistant" {
				exchangeCount++
			} else if i == len(messages)-1 {
				break
			}
		}
	}

	return exchangeCount
}

// ------------------------------------------------------------------------------------------------------
// findStartIndex calculates the starting index to keep the last N exchanges
func findStartIndex(messages []Message, exchangeCount, exchangesToKeep int) int {
	keptExchanges := 0
	startIndex := 0

	for i := 0; i < len(messages); i++ {
		if messages[i].Role == "user" {
			if i+1 < len(messages) && messages[i+1].Role == "assistant" {
				keptExchanges++
				if keptExchanges == (exchangeCount - exchangesToKeep + 1) {
					startIndex = i
					break
				}
			}
		}
	}

	return startIndex
}

// ------------------------------------------------------------------------------------------------------
func (s *MemoryStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = make([]Message, 0)
}
