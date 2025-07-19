package internal

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

func TestApp_ParseLog(t *testing.T) {
	t.Run("valid heroku router log", func(t *testing.T) {
		app := &App{
			ParsedLogChan: make(chan *ParsedLog, 1),
		}
		
		logEntry := `2025-07-19T10:30:45.123456+00:00 heroku[router]: at=info method=GET path="/api/users" host=myapp.herokuapp.com request_id=req-123 fwd="192.168.1.1" dyno=web.1 connect=10ms service=150ms status=200 bytes=1024 protocol=https`
		
		result := app.ParseLog([]byte(logEntry))
		
		// Check that the result contains expected key-value pairs
		expectedFields := map[string]string{
			"timestamp": "2025-07-19T10:30:45.123456+00:00",
			"at":        "info",
			"method":    "GET",
			"path":      `"/api/users"`,
			"host":      "myapp.herokuapp.com",
			"request_id": "req-123",
			"fwd":       `"192.168.1.1"`,
			"dyno":      "web.1",
			"connect":   "10ms",
			"service":   "150ms",
			"status":    "200",
			"bytes":     "1024",
			"protocol":  "https",
		}
		
		for key, expectedValue := range expectedFields {
			if value, exists := result[key]; !exists {
				t.Errorf("Expected key %q not found in result", key)
			} else if value != expectedValue {
				t.Errorf("For key %q: expected %q, got %q", key, expectedValue, value)
			}
		}
		
		// Check that a ParsedLog was sent to the channel
		select {
		case parsedLog := <-app.ParsedLogChan:
			if parsedLog == nil {
				t.Error("Expected ParsedLog to be sent to channel, but got nil")
			} else {
				// Verify some fields from the parsed log
				if parsedLog.Method != "GET" {
					t.Errorf("Expected method GET, got %q", parsedLog.Method)
				}
				if parsedLog.Status != 200 {
					t.Errorf("Expected status 200, got %d", parsedLog.Status)
				}
			}
		default:
			t.Error("Expected ParsedLog to be sent to channel, but channel is empty")
		}
	})

	t.Run("log with quoted values", func(t *testing.T) {
		app := &App{
			ParsedLogChan: make(chan *ParsedLog, 1),
		}
		
		logEntry := `2025-07-19T10:30:45.123456+00:00 heroku[router]: at=info method=POST path="/api/users/create" host=example.com fwd="203.0.113.1" status=201`
		
		result := app.ParseLog([]byte(logEntry))
		
		if result["path"] != `"/api/users/create"` {
			t.Errorf("Expected path %q, got %q", `"/api/users/create"`, result["path"])
		}
		
		if result["fwd"] != `"203.0.113.1"` {
			t.Errorf("Expected fwd %q, got %q", `"203.0.113.1"`, result["fwd"])
		}
		
		// Consume the channel
		<-app.ParsedLogChan
	})

	t.Run("log with special characters in path", func(t *testing.T) {
		app := &App{
			ParsedLogChan: make(chan *ParsedLog, 1),
		}
		
		logEntry := `2025-07-19T10:30:45.123456+00:00 heroku[router]: at=info method=GET path="/api/users?id=123&sort=name" host=example.com status=200`
		
		result := app.ParseLog([]byte(logEntry))
		
		if result["path"] != `"/api/users?id=123&sort=name"` {
			t.Errorf("Expected path with query params, got %q", result["path"])
		}
		
		// Consume the channel
		<-app.ParsedLogChan
	})

	t.Run("malformed log - no space", func(t *testing.T) {
		app := &App{
			ParsedLogChan: make(chan *ParsedLog, 1),
		}
		
		// Capture log output
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(os.Stderr)
		
		logEntry := `malformed_single_string_no_spaces`
		
		result := app.ParseLog([]byte(logEntry))
		
		// Should return empty map for malformed log
		if len(result) != 0 {
			t.Errorf("Expected empty map for malformed log, got %v", result)
		}
		
		// Check that error was logged
		logOutput := buf.String()
		if !strings.Contains(logOutput, "Malformed Request Received") {
			t.Error("Expected malformed request log message")
		}
		
		// Channel should be empty since no valid log was parsed
		select {
		case <-app.ParsedLogChan:
			t.Error("Expected no ParsedLog to be sent for malformed log")
		default:
			// Expected - channel should be empty
		}
	})

	t.Run("empty log", func(t *testing.T) {
		app := &App{
			ParsedLogChan: make(chan *ParsedLog, 1),
		}
		
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(os.Stderr)
		
		result := app.ParseLog([]byte(""))
		
		if len(result) != 0 {
			t.Errorf("Expected empty map for empty log, got %v", result)
		}
		
		logOutput := buf.String()
		if !strings.Contains(logOutput, "Malformed Request Received") {
			t.Error("Expected malformed request log message")
		}
	})

	t.Run("log with only timestamp", func(t *testing.T) {
		app := &App{
			ParsedLogChan: make(chan *ParsedLog, 1),
		}
		
		logEntry := `2025-07-19T10:30:45.123456+00:00 at=info`
		
		result := app.ParseLog([]byte(logEntry))
		
		expectedFields := map[string]string{
			"timestamp": "2025-07-19T10:30:45.123456+00:00",
			"at":        "info",
		}
		
		for key, expectedValue := range expectedFields {
			if value, exists := result[key]; !exists {
				t.Errorf("Expected key %q not found in result", key)
			} else if value != expectedValue {
				t.Errorf("For key %q: expected %q, got %q", key, expectedValue, value)
			}
		}
		
		// Consume the channel
		<-app.ParsedLogChan
	})

	t.Run("log with missing values", func(t *testing.T) {
		app := &App{
			ParsedLogChan: make(chan *ParsedLog, 1),
		}
		
		logEntry := `2025-07-19T10:30:45.123456+00:00 heroku[router]: at=info method= path="/api" status=`
		
		result := app.ParseLog([]byte(logEntry))
		
		// Fields with empty values should still be present
		if result["method"] != "" {
			t.Errorf("Expected empty method value, got %q", result["method"])
		}
		
		if result["status"] != "" {
			t.Errorf("Expected empty status value, got %q", result["status"])
		}
		
		// Consume the channel
		<-app.ParsedLogChan
	})

	t.Run("log with equals in value", func(t *testing.T) {
		app := &App{
			ParsedLogChan: make(chan *ParsedLog, 1),
		}
		
		// Test case where value contains equals sign
		logEntry := `2025-07-19T10:30:45.123456+00:00 heroku[router]: at=info path="/api?query=test=value" status=200`
		
		result := app.ParseLog([]byte(logEntry))
		
		// SplitN with limit 2 should handle this correctly
		if result["path"] != `"/api?query=test=value"` {
			t.Errorf("Expected path with equals in query, got %q", result["path"])
		}
		
		// Consume the channel
		<-app.ParsedLogChan
	})
}

func TestApp_ParserWorker(t *testing.T) {
	t.Run("processes multiple logs from channel", func(t *testing.T) {
		app := &App{
			RawLogChan:    make(chan []byte, 3),
			ParsedLogChan: make(chan *ParsedLog, 3),
		}
		
		// Add test logs to the channel
		testLogs := []string{
			`2025-07-19T10:30:45.123456+00:00 heroku[router]: at=info method=GET path="/api/users" status=200`,
			`2025-07-19T10:30:46.123456+00:00 heroku[router]: at=info method=POST path="/api/posts" status=201`,
			`2025-07-19T10:30:47.123456+00:00 heroku[router]: at=info method=DELETE path="/api/posts/123" status=204`,
		}
		
		for _, logEntry := range testLogs {
			app.RawLogChan <- []byte(logEntry)
		}
		close(app.RawLogChan) // Close channel to stop the worker
		
		// Start the worker in a goroutine
		done := make(chan bool)
		go func() {
			app.ParserWorker()
			done <- true
		}()
		
		// Wait for worker to finish
		<-done
		
		// Check that all logs were processed
		processedCount := 0
		for {
			select {
			case parsedLog := <-app.ParsedLogChan:
				if parsedLog != nil {
					processedCount++
				}
			default:
				goto checkCount
			}
		}
		
	checkCount:
		if processedCount != len(testLogs) {
			t.Errorf("Expected %d processed logs, got %d", len(testLogs), processedCount)
		}
	})

	t.Run("handles empty channel gracefully", func(t *testing.T) {
		app := &App{
			RawLogChan:    make(chan []byte),
			ParsedLogChan: make(chan *ParsedLog, 1),
		}
		
		// Close channel immediately
		close(app.RawLogChan)
		
		// Worker should exit gracefully when channel is closed
		done := make(chan bool)
		go func() {
			app.ParserWorker()
			done <- true
		}()
		
		// Should complete quickly
		select {
		case <-done:
			// Expected - worker should exit when channel is closed
		case <-time.After(100 * time.Millisecond):
			t.Error("ParserWorker did not exit when channel was closed")
		}
	})

	t.Run("processes logs continuously", func(t *testing.T) {
		app := &App{
			RawLogChan:    make(chan []byte, 1),
			ParsedLogChan: make(chan *ParsedLog, 10),
		}
		
		// Start worker
		go app.ParserWorker()
		
		// Send logs one by one
		testLogs := []string{
			`2025-07-19T10:30:45.123456+00:00 heroku[router]: at=info method=GET status=200`,
			`2025-07-19T10:30:46.123456+00:00 heroku[router]: at=info method=POST status=201`,
		}
		
		for _, logEntry := range testLogs {
			app.RawLogChan <- []byte(logEntry)
			
			// Give some time for processing
			time.Sleep(1 * time.Millisecond)
		}
		
		// Check that logs are being processed
		processedCount := 0
	outerLoop:
		for i := 0; i < len(testLogs); i++ {
			select {
			case parsedLog := <-app.ParsedLogChan:
				if parsedLog != nil {
					processedCount++
				}
			case <-time.After(50 * time.Millisecond):
				break outerLoop
			}
		}
		
		if processedCount != len(testLogs) {
			t.Errorf("Expected %d processed logs, got %d", len(testLogs), processedCount)
		}
		
		// Clean up
		close(app.RawLogChan)
	})
}

// Test helper function to validate the structure of parsed results
func TestParseLogStructure(t *testing.T) {
	t.Run("check field extraction accuracy", func(t *testing.T) {
		app := &App{
			ParsedLogChan: make(chan *ParsedLog, 1),
		}
		
		logEntry := `2025-07-19T10:30:45.123456+00:00 heroku[router]: at=info method=GET path="/test" host=example.com request_id=abc123 fwd="1.2.3.4" dyno=web.1 connect=5ms service=100ms status=200 bytes=512 protocol=https`
		
		result := app.ParseLog([]byte(logEntry))
		
		// Test that all expected fields are present
		requiredFields := []string{"timestamp", "at", "method", "path", "host", "request_id", "fwd", "dyno", "connect", "service", "status", "bytes", "protocol"}
		
		for _, field := range requiredFields {
			if _, exists := result[field]; !exists {
				t.Errorf("Required field %q missing from parsed result", field)
			}
		}
		
		// Test field values
		expectedValues := map[string]string{
			"timestamp":  "2025-07-19T10:30:45.123456+00:00",
			"at":         "info",
			"method":     "GET",
			"path":       `"/test"`,
			"host":       "example.com",
			"request_id": "abc123",
			"fwd":        `"1.2.3.4"`,
			"dyno":       "web.1",
			"connect":    "5ms",
			"service":    "100ms",
			"status":     "200",
			"bytes":      "512",
			"protocol":   "https",
		}
		
		for field, expected := range expectedValues {
			if actual := result[field]; actual != expected {
				t.Errorf("Field %q: expected %q, got %q", field, expected, actual)
			}
		}
		
		// Consume the channel
		<-app.ParsedLogChan
	})
}

// Benchmark tests
func BenchmarkApp_ParseLog(b *testing.B) {
	app := &App{
		ParsedLogChan: make(chan *ParsedLog, 1000),
	}
	
	logEntry := []byte(`2025-07-19T10:30:45.123456+00:00 heroku[router]: at=info method=GET path="/api/users" host=myapp.herokuapp.com request_id=req-123 fwd="192.168.1.1" dyno=web.1 connect=10ms service=150ms status=200 bytes=1024 protocol=https`)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.ParseLog(logEntry)
		// Consume from channel to prevent blocking
		<-app.ParsedLogChan
	}
}

func BenchmarkParseLogFields(b *testing.B) {
	app := &App{
		ParsedLogChan: make(chan *ParsedLog, 1000),
	}
	
	// Test with varying number of fields
	testCases := []struct {
		name string
		log  []byte
	}{
		{"minimal", []byte(`2025-07-19T10:30:45.123456+00:00 at=info status=200`)},
		{"medium", []byte(`2025-07-19T10:30:45.123456+00:00 heroku[router]: at=info method=GET path="/api" status=200 bytes=1024`)},
		{"full", []byte(`2025-07-19T10:30:45.123456+00:00 heroku[router]: at=info method=GET path="/api/users" host=myapp.herokuapp.com request_id=req-123 fwd="192.168.1.1" dyno=web.1 connect=10ms service=150ms status=200 bytes=1024 protocol=https`)},
	}
	
	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				app.ParseLog(tc.log)
				<-app.ParsedLogChan
			}
		})
	}
}

// Test edge cases and error conditions
func TestParseLogEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		description string
	}{
		{
			name:        "unicode characters",
			input:       `2025-07-19T10:30:45.123456+00:00 heroku[router]: at=info method=GET path="/api/测试" status=200`,
			expectError: false,
			description: "should handle unicode characters in values",
		},
		{
			name:        "very long log line",
			input:       `2025-07-19T10:30:45.123456+00:00 heroku[router]: at=info method=GET path="/api/` + strings.Repeat("x", 1000) + `" status=200`,
			expectError: false,
			description: "should handle very long log lines",
		},
		{
			name:        "log with newlines",
			input:       "2025-07-19T10:30:45.123456+00:00 heroku[router]: at=info method=GET\npath=\"/api\" status=200",
			expectError: false,
			description: "should handle logs with embedded newlines",
		},
		{
			name:        "multiple equals in value",
			input:       `2025-07-19T10:30:45.123456+00:00 heroku[router]: at=info path="/api?a=b=c=d" status=200`,
			expectError: false,
			description: "should handle multiple equals signs in values",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{
				ParsedLogChan: make(chan *ParsedLog, 1),
			}
			
			result := app.ParseLog([]byte(tt.input))
			
			if tt.expectError {
				if len(result) != 0 {
					t.Errorf("%s: expected error but got result: %v", tt.description, result)
				}
			} else {
				if len(result) == 0 {
					t.Errorf("%s: expected result but got empty map", tt.description)
				}
				// Always should have timestamp if not an error case
				if result["timestamp"] == "" {
					t.Errorf("%s: expected timestamp in result", tt.description)
				}
			}
			
			// Consume channel if not empty
			select {
			case <-app.ParsedLogChan:
			default:
			}
		})
	}
}
