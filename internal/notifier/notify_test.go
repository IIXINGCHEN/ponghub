package notifier

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wcy-dt/ponghub/internal/types/structures/checker"
	"github.com/wcy-dt/ponghub/internal/types/types/chk_result"
)

//goland:noinspection HttpUrlsUsage,HttpUrlsUsage,HttpUrlsUsage,HttpUrlsUsage,HttpUrlsUsage
func TestCollectUnavailableEndpoints(t *testing.T) {
	//goland:noinspection ALL
	checkResult := []checker.Service{
		{
			Name: "Service1",
			Endpoints: []checker.Endpoint{
				{URL: "http://example.com", Status: chk_result.NONE},
				{URL: "http://good.com", Status: chk_result.ALL},
			},
		},
		{
			Name: "Service2",
			Endpoints: []checker.Endpoint{
				{URL: "http://bad.com", Status: chk_result.NONE},
			},
		},
	}

	result := collectUnavailableEndpoints(checkResult)

	if len(result) != 2 {
		t.Errorf("Expected 2 services with unavailable endpoints, got %d", len(result))
	}

	if len(result["Service1"]) != 1 {
		t.Errorf("Expected 1 unavailable endpoint for Service1, got %d", len(result["Service1"]))
	}

	if len(result["Service2"]) != 1 {
		t.Errorf("Expected 1 unavailable endpoint for Service2, got %d", len(result["Service2"]))
	}

	if result["Service1"][0].URL != "http://example.com" {
		t.Errorf("Expected URL http://example.com, got %s", result["Service1"][0].URL)
	}
}

//goland:noinspection HttpUrlsUsage
func TestCollectCertProblemEndpoints(t *testing.T) {
	checkResult := []checker.Service{
		{
			Name: "Service1",
			Endpoints: []checker.Endpoint{
				{
					URL:               "https://expired.com",
					IsHTTPS:           true,
					IsCertExpired:     true,
					CertRemainingDays: -1,
				},
				{
					URL:               "https://expiring.com",
					IsHTTPS:           true,
					IsCertExpired:     false,
					CertRemainingDays: 5,
				},
				{
					URL:               "https://good.com",
					IsHTTPS:           true,
					IsCertExpired:     false,
					CertRemainingDays: 30,
				},
				{
					URL:     "http://notssl.com",
					IsHTTPS: false,
				},
			},
		},
	}

	certNotifyDays := 7
	result := collectCertProblemEndpoints(checkResult, certNotifyDays)

	if len(result) != 1 {
		t.Errorf("Expected 1 service with cert problems, got %d", len(result))
	}

	if len(result["Service1"]) != 2 {
		t.Errorf("Expected 2 endpoints with cert problems, got %d", len(result["Service1"]))
	}

	// Check if both expired and soon-to-expire endpoints are collected
	urls := []string{result["Service1"][0].URL, result["Service1"][1].URL}
	expectedURLs := []string{"https://expired.com", "https://expiring.com"}

	for _, expectedURL := range expectedURLs {
		found := false
		for _, url := range urls {
			if url == expectedURL {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find URL %s in cert problem endpoints", expectedURL)
		}
	}
}

//goland:noinspection HttpUrlsUsage,HttpUrlsUsage,HttpUrlsUsage
func TestCountEndpoints(t *testing.T) {
	endpointsMap := map[string][]checker.Endpoint{
		"Service1": {
			{URL: "http://example1.com"},
			{URL: "http://example2.com"},
		},
		"Service2": {
			{URL: "http://example3.com"},
		},
	}

	count := countEndpoints(endpointsMap)
	expectedCount := 3

	if count != expectedCount {
		t.Errorf("Expected count %d, got %d", expectedCount, count)
	}
}

func TestRemoveExistingNotifyFile(t *testing.T) {
	// Create a temporary file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_notify.txt")

	// Create the file
	f, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Failed to close test file: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatal("Test file should exist")
	}

	// Remove the file
	err = removeExistingNotifyFile(testFile)
	if err != nil {
		t.Errorf("removeExistingNotifyFile failed: %v", err)
	}

	// Verify file is removed
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("Test file should be removed")
	}

	// Test removing non-existent file (should not error)
	err = removeExistingNotifyFile(testFile)
	if err != nil {
		t.Errorf("removeExistingNotifyFile should not error on non-existent file: %v", err)
	}
}

func TestWriteToFile(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_write.txt")

	f, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Errorf("Failed to close test file: %v", err)
		}
	}()

	testContent := "Hello, World!"
	writeToFile(f, testContent)

	// Read back the content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("Expected content %q, got %q", testContent, string(content))
	}
}

//goland:noinspection HttpUrlsUsage,HttpUrlsUsage
func TestWriteNotificationReport(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_report.txt")

	f, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Errorf("Failed to close test file: %v", err)
		}
	}()

	// Create test data
	statusNoneEndpoints := map[string][]checker.Endpoint{
		"TestService": {
			{
				URL:            "http://test.com",
				Method:         "GET",
				StatusCode:     500,
				ResponseTime:   100 * time.Millisecond,
				AttemptNum:     3,
				SuccessNum:     0,
				StartTime:      "2025-01-01 10:00:00",
				EndTime:        "2025-01-01 10:00:01",
				FailureDetails: []string{"Connection timeout", "Server error"},
				ResponseBody:   "Internal Server Error",
			},
		},
	}

	certProblemEndpoints := map[string][]checker.Endpoint{
		"SSLService": {
			{
				URL:               "https://ssl.com",
				IsHTTPS:           true,
				IsCertExpired:     true,
				CertRemainingDays: -5,
				StatusCode:        200,
				ResponseTime:      50 * time.Millisecond,
				StartTime:         "2025-01-01 10:00:00",
				EndTime:           "2025-01-01 10:00:01",
			},
		},
	}

	writeNotificationReport(f, statusNoneEndpoints, certProblemEndpoints)

	// Read back the content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	contentStr := string(content)

	// Check for expected sections
	expectedSections := []string{
		"=== PongHub Service Status Report ===",
		"🔴 UNAVAILABLE SERVICES:",
		"📋 Service: TestService",
		"• URL: http://test.com",
		"Method: GET",
		"Status Code: 500",
		"Response Time: 100ms",
		"Attempts: 0/3 successful",
		"Failure Details:",
		"Connection timeout",
		"Server error",
		"Response Body: Internal Server Error",
		"🔐 CERTIFICATE ISSUES:",
		"📋 Service: SSLService",
		"• URL: https://ssl.com",
		"❌ Certificate Status: EXPIRED",
		"Days Remaining: -5",
		"📊 SUMMARY:",
		"Unavailable Endpoints: 1",
		"Certificate Issues: 1",
		"Total Issues: 2",
	}

	for _, section := range expectedSections {
		if !strings.Contains(contentStr, section) {
			t.Errorf("Expected to find section %q in report", section)
		}
	}
}

func TestWriteCertificateStatus(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_cert_status.txt")

	f, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Errorf("Failed to close test file: %v", err)
		}
	}()

	tests := []struct {
		name     string
		endpoint checker.Endpoint
		expected string
	}{
		{
			name: "Expired certificate",
			endpoint: checker.Endpoint{
				IsCertExpired:     true,
				CertRemainingDays: -1,
			},
			expected: "❌ Certificate Status: EXPIRED",
		},
		{
			name: "Certificate expires in 1 day",
			endpoint: checker.Endpoint{
				IsCertExpired:     false,
				CertRemainingDays: 1,
			},
			expected: "🚨 Certificate Status: EXPIRES IN 1 DAY OR LESS",
		},
		{
			name: "Certificate expires soon",
			endpoint: checker.Endpoint{
				IsCertExpired:     false,
				CertRemainingDays: 5,
			},
			expected: "⚠️  Certificate Status: EXPIRES SOON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear file content
			if err := f.Truncate(0); err != nil {
				t.Fatalf("Failed to truncate file: %v", err)
			}
			if _, err := f.Seek(0, 0); err != nil {
				t.Fatalf("Failed to seek file: %v", err)
			}

			writeCertificateStatus(f, tt.endpoint)

			content, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatalf("Failed to read test file: %v", err)
			}

			if !strings.Contains(string(content), tt.expected) {
				t.Errorf("Expected content to contain %q, got %q", tt.expected, string(content))
			}
		})
	}
}

func TestWriteFailureDetails(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_failure.txt")

	f, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Errorf("Failed to close test file: %v", err)
		}
	}()

	// Test with failure details
	failureDetails := []string{"Error 1", "Error 2"}
	writeFailureDetails(f, failureDetails)

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "Failure Details:") {
		t.Error("Expected to find 'Failure Details:' header")
	}
	if !strings.Contains(contentStr, "- Error 1") {
		t.Error("Expected to find '- Error 1'")
	}
	if !strings.Contains(contentStr, "- Error 2") {
		t.Error("Expected to find '- Error 2'")
	}

	// Test with empty failure details
	if err := f.Truncate(0); err != nil {
		t.Fatalf("Failed to truncate file: %v", err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatalf("Failed to seek file: %v", err)
	}
	writeFailureDetails(f, []string{})

	content, err = os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	if len(content) != 0 {
		t.Error("Expected no content for empty failure details")
	}
}

func TestWriteResponseBody(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_response.txt")

	f, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Errorf("Failed to close test file: %v", err)
		}
	}()

	// Test with short response body
	shortBody := "Short response"
	writeResponseBody(f, shortBody)

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	if !strings.Contains(string(content), shortBody) {
		t.Errorf("Expected to find response body %q", shortBody)
	}

	// Test with long response body (should be skipped)
	if err := f.Truncate(0); err != nil {
		t.Fatalf("Failed to truncate file: %v", err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatalf("Failed to seek file: %v", err)
	}
	longBody := strings.Repeat("x", 600) // More than 500 chars
	writeResponseBody(f, longBody)

	content, err = os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	if len(content) != 0 {
		t.Error("Expected no content for long response body")
	}

	// Test with empty response body
	if err := f.Truncate(0); err != nil {
		t.Fatalf("Failed to truncate file: %v", err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatalf("Failed to seek file: %v", err)
	}
	writeResponseBody(f, "")

	content, err = os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	if len(content) != 0 {
		t.Error("Expected no content for empty response body")
	}
}
