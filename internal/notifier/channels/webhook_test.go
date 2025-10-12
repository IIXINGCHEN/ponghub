package channels

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/wcy-dt/ponghub/internal/types/structures/configure"
)

// TestWebhookNotifier_BasicSend tests basic webhook functionality
//
//goland:noinspection DuplicatedCode
func TestWebhookNotifier_BasicSend(t *testing.T) {
	// Create a test server to receive webhook
	var receivedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Failed to read request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Parse JSON
		if err := json.Unmarshal(body, &receivedPayload); err != nil {
			t.Errorf("Failed to parse JSON: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("OK"))
		if err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	// Create webhook config
	config := &configure.WebhookConfig{
		URL:    server.URL,
		Method: "POST",
	}

	// Create notifier and send
	notifier := NewWebhookNotifier(config)
	err := notifier.Send("Test Alert", "This is a test message")

	if err != nil {
		t.Fatalf("Failed to send webhook: %v", err)
	}

	// Verify received data
	if receivedPayload["title"] != "Test Alert" {
		t.Errorf("Expected title 'Test Alert', got '%v'", receivedPayload["title"])
	}

	if receivedPayload["message"] != "This is a test message" {
		t.Errorf("Expected message 'This is a test message', got '%v'", receivedPayload["message"])
	}

	if receivedPayload["service"] != "ponghub" {
		t.Errorf("Expected service 'ponghub', got '%v'", receivedPayload["service"])
	}
}

// TestWebhookNotifier_CustomPayload tests the custom payload functionality
//
//goland:noinspection DuplicatedCode
func TestWebhookNotifier_CustomPayload(t *testing.T) {
	// Test server to capture requests
	var receivedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Read body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Failed to read request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Parse JSON
		if err := json.Unmarshal(body, &receivedPayload); err != nil {
			t.Errorf("Failed to parse JSON: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create webhook config with custom payload
	config := &configure.WebhookConfig{
		URL:    server.URL,
		Method: "POST",
		CustomPayload: &configure.CustomPayloadConfig{
			Template:    `{"alert": "{{.Title}}", "details": "{{.Message}}", "env": "{{.environment}}"}`,
			ContentType: "application/json",
			Fields: map[string]string{
				"environment": "production",
			},
		},
	}

	notifier := NewWebhookNotifier(config)
	err := notifier.Send("Service Down", "Database connection failed")

	if err != nil {
		t.Fatalf("Failed to send webhook: %v", err)
	}

	// Verify the custom template was used correctly
	if receivedPayload["alert"] != "Service Down" {
		t.Errorf("Expected alert 'Service Down', got '%v'", receivedPayload["alert"])
	}

	if receivedPayload["details"] != "Database connection failed" {
		t.Errorf("Expected details 'Database connection failed', got '%v'", receivedPayload["details"])
	}

	if receivedPayload["env"] != "production" {
		t.Errorf("Expected env 'production', got '%v'", receivedPayload["env"])
	}
}

// TestWebhookNotifier_Authentication tests Bearer token authentication
func TestWebhookNotifier_Authentication(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &configure.WebhookConfig{
		URL:       server.URL,
		Method:    "POST",
		AuthType:  "bearer",
		AuthToken: "test-token-123",
	}

	notifier := NewWebhookNotifier(config)
	err := notifier.Send("Test", "Test message")

	if err != nil {
		t.Fatalf("Failed to send webhook: %v", err)
	}

	expectedAuth := "Bearer test-token-123"
	if receivedAuth != expectedAuth {
		t.Errorf("Expected Authorization '%s', got '%s'", expectedAuth, receivedAuth)
	}
}

// TestWebhookNotifier_ErrorHandling tests error handling and retries
func TestWebhookNotifier_ErrorHandling(t *testing.T) {
	// Test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte("Internal Server Error"))
		if err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	config := &configure.WebhookConfig{
		URL:     server.URL,
		Method:  "POST",
		Retries: 0, // No retries for this test
	}

	notifier := NewWebhookNotifier(config)
	err := notifier.Send("Test Alert", "Test message")

	if err == nil {
		t.Fatal("Expected error for 500 status code")
	}

	requestCount := 0
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config = &configure.WebhookConfig{
		URL:     server.URL,
		Method:  "POST",
		Retries: 3,
		Timeout: 5,
	}

	notifier = NewWebhookNotifier(config)
	err = notifier.Send("Test Alert", "Test message")

	if err != nil {
		t.Fatalf("Expected success after retries, got error: %v", err)
	}

	if requestCount != 3 {
		t.Errorf("Expected 3 requests (2 retries + 1 success), got %d", requestCount)
	}
}

// TestWebhookNotifier_PresetFormats tests preset format (Slack)
func TestWebhookNotifier_PresetFormats(t *testing.T) {
	var receivedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Failed to read request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if err := json.Unmarshal(body, &receivedPayload); err != nil {
			t.Errorf("Failed to parse JSON: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &configure.WebhookConfig{
		URL:    server.URL,
		Method: "POST",
		Format: "slack",
	}

	notifier := NewWebhookNotifier(config)
	err := notifier.Send("Service Alert", "Service is down")

	if err != nil {
		t.Fatalf("Failed to send webhook: %v", err)
	}

	// Verify Slack format structure
	if receivedPayload["text"] != "*Service Alert*" {
		t.Errorf("Expected text '*Service Alert*', got '%v'", receivedPayload["text"])
	}

	attachments, ok := receivedPayload["attachments"].([]interface{})
	if !ok || len(attachments) == 0 {
		t.Fatal("Expected attachments array")
	}

	attachment := attachments[0].(map[string]interface{})
	if attachment["text"] != "Service is down" {
		t.Errorf("Expected attachment text 'Service is down', got '%v'", attachment["text"])
	}
}

// TestWebhookNotifier_ConcurrentRequests tests concurrent webhook sending
func TestWebhookNotifier_ConcurrentRequests(t *testing.T) {
	var requestCount int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&requestCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &configure.WebhookConfig{
		URL:    server.URL,
		Method: "POST",
	}

	notifier := NewWebhookNotifier(config)

	const numWorkers = 5
	const requestsPerWorker = 2

	errChan := make(chan error, numWorkers*requestsPerWorker)

	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			for j := 0; j < requestsPerWorker; j++ {
				err := notifier.Send("Test", "Message")
				errChan <- err
			}
		}(i)
	}

	// Collect results
	var errors []error
	for i := 0; i < numWorkers*requestsPerWorker; i++ {
		if err := <-errChan; err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		t.Fatalf("Got %d errors from concurrent requests: %v", len(errors), errors[0])
	}

	finalCount := atomic.LoadInt64(&requestCount)
	expectedCount := int64(numWorkers * requestsPerWorker)
	if finalCount != expectedCount {
		t.Errorf("Expected %d requests, got %d", expectedCount, finalCount)
	}
}

// TestWebhookNotifier_RealWorldScenario tests real-world webhook usage
func TestWebhookNotifier_RealWorldScenario(t *testing.T) {
	type AlertPayload struct {
		Alert   string `json:"alert"`
		Details string `json:"details"`
	}

	var receivedAlert AlertPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if err := json.NewDecoder(r.Body).Decode(&receivedAlert); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"status": "success"}`))
		if err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	config := &configure.WebhookConfig{
		URL:    server.URL,
		Method: "POST",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		CustomPayload: &configure.CustomPayloadConfig{
			Template: `{"alert": "{{.Title}}", "details": "{{.Message}}"}`,
		},
	}

	notifier := NewWebhookNotifier(config)

	title := "🔴 PongHub Service Status Alert"
	message := "Generated at: 2025-10-12 10:00:00\n\nService check failed"

	err := notifier.Send(title, message)
	if err != nil {
		t.Fatalf("Failed to send real-world webhook: %v", err)
	}

	if receivedAlert.Alert != title {
		t.Errorf("Expected alert field '%s', got '%s'", title, receivedAlert.Alert)
	}

	if receivedAlert.Details != message {
		t.Errorf("Expected details field '%s', got '%s'", message, receivedAlert.Details)
	}
}
