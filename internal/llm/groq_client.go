package llm

import (
	"bufio"
	"encoding/json"
	"net/http"
	"time"

	apperror "llm-chat-service/internal/error"
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

// ------------------------------------------------------------------------------------------------------
func (c *GroqClient) StreamChat(messages []Message, maxTokens int, onToken func(string) error) (string, error) {
	reqBody := ChatRequest{
		Model:     c.model,
		Messages:  messages,
		Stream:    true,
		MaxTokens: maxTokens,
	}

	resp, err := c.DoRequest(reqBody)
	if err != nil {
		return "", err 
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)

	fullResponse, err := ScanStream(scanner, onToken)
	if err != nil {
		return "", apperror.NewLLMError("failed to process LLM stream", err)
	}

	return fullResponse.String(), nil
}

// ------------------------------------------------------------------------------------------------------
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
		return "", err // Already wrapped with AppError
	}
	defer resp.Body.Close()

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", apperror.NewLLMError("failed to decode LLM API response", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", apperror.NewLLMError("no choices in LLM response", nil)
	}

	choice := chatResp.Choices[0]
	if choice.Message == nil {
		return "", apperror.NewLLMError("message is nil in LLM response choice", nil)
	}

	content := choice.Message.Content
	if content == "" {
		return "", apperror.NewLLMError("empty content in LLM response", nil)
	}

	return content, nil
}
