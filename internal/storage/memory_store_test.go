package storage

import (
	"testing"
)

func TestMemoryStore_AddMessage(t *testing.T) {
	store := NewMemoryStore(20)

	msg := Message{Role: "user", Content: "Hello"}
	store.AddMessage(msg)

	messages := store.GetMessages()
	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}
	if messages[0].Content != "Hello" {
		t.Errorf("Expected 'Hello', got '%s'", messages[0].Content)
	}
}

func TestMemoryStore_TrimToMaxExchanges(t *testing.T) {
	store := NewMemoryStore(2) // Keep only 2 exchanges

	// Add 3 exchanges (6 messages)
	for i := 0; i < 3; i++ {
		store.AddMessage(Message{Role: "user", Content: "Question"})
		store.AddMessage(Message{Role: "assistant", Content: "Answer"})
	}

	messages := store.GetMessages()
	// Should keep only last 2 exchanges (4 messages)
	if len(messages) > 4 {
		t.Errorf("Expected at most 4 messages, got %d", len(messages))
	}
}

func TestMemoryStore_Concurrency(t *testing.T) {
	store := NewMemoryStore(20)

	// Test concurrent access
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(id int) {
			store.AddMessage(Message{Role: "user", Content: "Message"})
			_ = store.GetMessages()
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	messages := store.GetMessages()
	if len(messages) != 10 {
		t.Errorf("Expected 10 messages, got %d", len(messages))
	}
}

func TestMemoryStore_Clear(t *testing.T) {
	store := NewMemoryStore(20)

	store.AddMessage(Message{Role: "user", Content: "Hello"})
	store.Clear()

	messages := store.GetMessages()
	if len(messages) != 0 {
		t.Errorf("Expected 0 messages after clear, got %d", len(messages))
	}
}
