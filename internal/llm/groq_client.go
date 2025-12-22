package llm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client interface for LLM operations
type Client interface {
	Chat(messages []Message, maxTokens int) (string, error)
	StreamChat(messages []Message, maxTokens int, onToken func(string) error) (string, error)
}

// GroqClient handles communication with Groq API
type GroqClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	model      string
}

// NewGroqClient creates a new Groq client
func NewGroqClient(apiKey string, baseURL string, model string) *GroqClient {
	return &GroqClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		model: model,
	}
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents the request to Groq API
type ChatRequest struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	Stream    bool      `json:"stream"`
	MaxTokens int       `json:"max_tokens,omitempty"`
}

// ChatResponse represents a streaming response chunk
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
}

// Choice represents a choice in the response
type Choice struct {
	Index        int      `json:"index"`
	Delta        *Delta   `json:"delta,omitempty"`
	Message      *Message `json:"message,omitempty"`
	FinishReason string   `json:"finish_reason,omitempty"`
}

// Delta represents incremental content in streaming
type Delta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// StreamChat streams the chat completion response
func (c *GroqClient) StreamChat(messages []Message, maxTokens int, onToken func(string) error) (string, error) {
	reqBody := ChatRequest{
		Model:     c.model,
		Messages:  messages,
		Stream:    true,
		MaxTokens: maxTokens,
	}

	resp, err := c.DoRequest(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)

	fullResponse, err := ScanStream(scanner, onToken)
	if err != nil {
		return "", fmt.Errorf("failed to scan stream: %w", err)
	}

	return fullResponse.String(), nil
}

// Chat performs a non-streaming chat completion
func (c *GroqClient) Chat(messages []Message, maxTokens int) (string, error) {
	reqBody := ChatRequest{
		Model:     c.model,
		Messages:  messages,
		Stream:    false,
		MaxTokens: maxTokens,
	}

	resp, err := c.DoRequest(reqBody)
	if err != nil {
		return "", fmt.Errorf("groq API error: %w", err)
	}
	defer resp.Body.Close()

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("groq API error: failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("groq API error: no choices in response")
	}

	choice := chatResp.Choices[0]
	if choice.Message == nil {
		return "", fmt.Errorf("groq API error: message is nil in response choice")
	}

	content := choice.Message.Content
	if content == "" {
		return "", fmt.Errorf("groq API error: empty content in response")
	}

	return content, nil
}
