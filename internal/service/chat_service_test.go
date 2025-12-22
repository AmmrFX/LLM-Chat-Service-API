package service

import (
	"errors"
	"llm-chat-service/internal/llm"
	"llm-chat-service/internal/storage"
	"testing"
)

// Mock GroqClient for testing
type mockGroqClient struct {
	chatFunc       func([]llm.Message, int) (string, error)
	streamChatFunc func([]llm.Message, int, func(string) error) (string, error)
}

func (m *mockGroqClient) Chat(messages []llm.Message, maxTokens int) (string, error) {
	if m.chatFunc != nil {
		return m.chatFunc(messages, maxTokens)
	}
	return "mock response", nil
}

func (m *mockGroqClient) StreamChat(messages []llm.Message, maxTokens int, onToken func(string) error) (string, error) {
	if m.streamChatFunc != nil {
		return m.streamChatFunc(messages, maxTokens, onToken)
	}
	// Default behavior: call onToken with response
	if onToken != nil {
		_ = onToken("mock")
		_ = onToken(" stream")
		_ = onToken(" response")
	}
	return "mock stream response", nil
}

func TestChatRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request ChatRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: ChatRequest{
				Messages: []storage.Message{
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr: false,
		},
		{
			name: "empty messages",
			request: ChatRequest{
				Messages: []storage.Message{},
			},
			wantErr: true,
		},
		{
			name: "invalid role",
			request: ChatRequest{
				Messages: []storage.Message{
					{Role: "invalid", Content: "Hello"},
				},
			},
			wantErr: true,
		},
		{
			name: "empty content",
			request: ChatRequest{
				Messages: []storage.Message{
					{Role: "user", Content: ""},
				},
			},
			wantErr: true,
		},
		{
			name: "last message not from user",
			request: ChatRequest{
				Messages: []storage.Message{
					{Role: "assistant", Content: "Hello"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChatService_ProcessChat(t *testing.T) {
	memoryStore := storage.NewMemoryStore(20)
	mockClient := &mockGroqClient{
		chatFunc: func(messages []llm.Message, maxTokens int) (string, error) {
			return "test response", nil
		},
	}

	service := NewChatService(memoryStore, nil, mockClient, 1024)

	req := &ChatRequest{
		Messages: []storage.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	response, err := service.ProcessChat(req)
	if err != nil {
		t.Errorf("ProcessChat() error = %v", err)
	}
	if response != "test response" {
		t.Errorf("ProcessChat() response = %v, want 'test response'", response)
	}

	// Check that message was added to history
	messages := memoryStore.GetMessages()
	if len(messages) != 2 { // user message + assistant response
		t.Errorf("Expected 2 messages in history, got %d", len(messages))
	}
}

func TestChatService_ProcessChat_Error(t *testing.T) {
	memoryStore := storage.NewMemoryStore(20)
	mockClient := &mockGroqClient{
		chatFunc: func(messages []llm.Message, maxTokens int) (string, error) {
			return "", errors.New("API error")
		},
	}

	service := NewChatService(memoryStore, nil, mockClient, 1024)

	req := &ChatRequest{
		Messages: []storage.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	_, err := service.ProcessChat(req)
	if err == nil {
		t.Error("ProcessChat() expected error, got nil")
	}
}

func TestChatService_ProcessChatStream(t *testing.T) {
	memoryStore := storage.NewMemoryStore(20)
	tokens := []string{}

	mockClient := &mockGroqClient{
		streamChatFunc: func(messages []llm.Message, maxTokens int, onToken func(string) error) (string, error) {
			// Simulate streaming tokens
			onToken("Hello")
			onToken(" World")
			return "Hello World", nil
		},
	}

	service := NewChatService(memoryStore, nil, mockClient, 1024)

	req := &ChatRequest{
		Messages: []storage.Message{
			{Role: "user", Content: "Hello"},
		},
		Stream: true,
	}

	response, err := service.ProcessChatStream(req, func(token string) error {
		tokens = append(tokens, token)
		return nil
	})

	if err != nil {
		t.Errorf("ProcessChatStream() error = %v", err)
	}
	if response != "Hello World" {
		t.Errorf("ProcessChatStream() response = %v, want 'Hello World'", response)
	}
	if len(tokens) != 2 {
		t.Errorf("Expected 2 tokens, got %d", len(tokens))
	}
}
