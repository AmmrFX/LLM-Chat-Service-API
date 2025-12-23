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

	apperror "llm-chat-service/internal/error"
)

func (c *GroqClient) DoRequest(reqBody any) (*http.Response, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, apperror.NewInternalError("failed to marshal LLM request", err)
	}

	req, err := http.NewRequest("POST", c.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, apperror.NewInternalError("failed to create HTTP request", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// Set a deadline for the request
	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Check if it's a timeout error
		if err.Error() == "context deadline exceeded" ||
			strings.Contains(err.Error(), "timeout") ||
			strings.Contains(err.Error(), "Client.Timeout exceeded") {
			return nil, apperror.NewTimeoutError("LLM API request timed out", err)
		}
		return nil, apperror.NewLLMError("failed to send request to LLM API", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Check for specific HTTP status codes
		switch resp.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return nil, apperror.NewUnauthorizedError(
				fmt.Sprintf("LLM API authentication failed (status %d)", resp.StatusCode),
				fmt.Errorf("response: %s", string(bodyBytes)),
			)
		case http.StatusTooManyRequests:
			return nil, apperror.NewRateLimitError(
				"LLM API rate limit exceeded",
				fmt.Errorf("status %d, response: %s", resp.StatusCode, string(bodyBytes)),
			)
		case http.StatusGatewayTimeout, http.StatusRequestTimeout:
			duration := time.Since(start)
			return nil, apperror.NewTimeoutError(
				fmt.Sprintf("LLM API timed out after %v", duration),
				fmt.Errorf("status %d", resp.StatusCode),
			)
		default:
			return nil, apperror.NewLLMError(
				fmt.Sprintf("LLM API returned error status %d", resp.StatusCode),
				fmt.Errorf("response: %s", string(bodyBytes)),
			)
		}
	}

	// Don't close the body here - let the caller close it after reading
	return resp, nil
}

func ScanStream(scanner *bufio.Scanner, onToken func(string) error) (strings.Builder, error) {
	var fullResponse strings.Builder
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
					return strings.Builder{}, err
				}
			}
		}
	}

	return fullResponse, nil
}
