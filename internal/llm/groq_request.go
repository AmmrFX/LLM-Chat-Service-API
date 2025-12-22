package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (c *GroqClient) DoRequest(reqBody any) (*http.Response, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("groq API error: status %d, body: %s", resp.StatusCode, string(bodyBytes))
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
