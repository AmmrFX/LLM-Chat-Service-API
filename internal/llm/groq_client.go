package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
}

// NewGroqClient creates a new Groq client
func NewGroqClient(apiKey string) *GroqClient {
	return &GroqClient{
		apiKey:  apiKey,
		baseURL: "https://api.groq.com/openai/v1/chat/completions",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents the request to Groq API
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Stream      bool      `json:"stream"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
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
		Model:     "llama-3.1-8b-instant", // Free model available on Groq
		Messages:  messages,
		Stream:    true,
		MaxTokens: maxTokens,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("groq API error: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var fullResponse strings.Builder
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Skip SSE prefix "data: "
		if bytes.HasPrefix(line, []byte("data: ")) {
			line = line[6:]
		}

		// Check for [DONE] marker
		if bytes.Equal(line, []byte("[DONE]")) {
			break
		}

		var chatResp ChatResponse
		if err := json.Unmarshal(line, &chatResp); err != nil {
			continue
		}

		if len(chatResp.Choices) > 0 {
			choice := chatResp.Choices[0]
			var content string
			if choice.Delta != nil {
				content = choice.Delta.Content
			} else if choice.Message != nil {
				content = choice.Message.Content
			}

			if content != "" {
				fullResponse.WriteString(content)
				if err := onToken(content); err != nil {
					return "", err
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read stream: %w", err)
	}

	return fullResponse.String(), nil
}

// Chat performs a non-streaming chat completion
func (c *GroqClient) Chat(messages []Message, maxTokens int) (string, error) {
	reqBody := ChatRequest{
		Model:     "llama-3.1-8b-instant",
		Messages:  messages,
		Stream:    false,
		MaxTokens: maxTokens,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("groq API error: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) > 0 && chatResp.Choices[0].Message != nil {
		return chatResp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no response content in API response")
}
