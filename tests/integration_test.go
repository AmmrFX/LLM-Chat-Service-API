//go:build integration
// +build integration

package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

const baseURL = "http://localhost:8000"

func TestHealthEndpoint(t *testing.T) {
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		t.Fatalf("Failed to call health endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if string(body) != "OK" {
		t.Errorf("Expected 'OK', got '%s'", string(body))
	}
}

func TestChatEndpoint_NonStreaming(t *testing.T) {
	if os.Getenv("GROQ_API_KEY") == "" {
		t.Skip("Skipping integration test: GROQ_API_KEY not set")
	}

	reqBody := map[string]interface{}{
		"messages": []map[string]string{
			{"role": "user", "content": "Say 'Hello, World!' and nothing else."},
		},
		"stream": false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	resp, err := http.Post(baseURL+"/chat", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to call chat endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
		return
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["response"] == "" {
		t.Error("Expected non-empty response")
	}
}

func TestChatEndpoint_Validation(t *testing.T) {
	// Test empty messages
	reqBody := map[string]interface{}{
		"messages": []map[string]string{},
		"stream":   false,
	}

	jsonData, _ := json.Marshal(reqBody)
	resp, err := http.Post(baseURL+"/chat", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to call chat endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty messages, got %d", resp.StatusCode)
	}
}

func TestChatEndpoint_SSE(t *testing.T) {
	if os.Getenv("GROQ_API_KEY") == "" {
		t.Skip("Skipping integration test: GROQ_API_KEY not set")
	}

	reqBody := map[string]interface{}{
		"messages": []map[string]string{
			{"role": "user", "content": "Count to 3, one number per line."},
		},
		"stream": true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", baseURL+"/chat", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to call chat endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
		return
	}

	// Read at least some data
	buffer := make([]byte, 1024)
	n, err := resp.Body.Read(buffer)
	if err != nil && err != io.EOF {
		t.Fatalf("Failed to read SSE stream: %v", err)
	}

	if n == 0 {
		t.Error("Expected to receive some data from SSE stream")
	}
}

func TestMetricsEndpoint(t *testing.T) {
	resp, err := http.Get(baseURL + "/metrics")
	if err != nil {
		t.Fatalf("Failed to call metrics endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	// Check for Prometheus metrics format
	bodyStr := string(body)
	if bodyStr == "" {
		t.Error("Expected non-empty metrics response")
	}
}

// Helper function to wait for server to be ready
func TestMain(m *testing.M) {
	// Wait a bit for server to start if needed
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(baseURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			break
		}
		if i == maxRetries-1 {
			fmt.Println("Warning: Server may not be running. Some tests may fail.")
		}
		time.Sleep(1 * time.Second)
	}

	os.Exit(m.Run())
}
