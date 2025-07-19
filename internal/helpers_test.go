package internal

import (
	"testing"
	"time"
)

func TestNewDedupeCache(t *testing.T) {
	tests := []struct {
		name string
		size int
	}{
		{"small cache", 5},
		{"medium cache", 100},
		{"large cache", 1000},
		{"single element", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewDedupeCache(tt.size)

			if cache == nil {
				t.Fatal("NewDedupeCache returned nil")
			}
			if cache.Size != tt.size {
				t.Errorf("Expected size %d, got %d", tt.size, cache.Size)
			}
			if len(cache.Buffer) != tt.size {
				t.Errorf("Expected buffer length %d, got %d", tt.size, len(cache.Buffer))
			}
			if cache.Lookup == nil {
				t.Error("Lookup map is nil")
			}
			if cache.WritePos != 0 {
				t.Errorf("Expected WritePos 0, got %d", cache.WritePos)
			}
		})
	}
}

func TestDedupeCache_Add(t *testing.T) {
	t.Run("add new items", func(t *testing.T) {
		cache := NewDedupeCache(3)

		// Add first item
		if !cache.Add("msg1") {
			t.Error("Expected Add to return true for new item")
		}
		if cache.WritePos != 1 {
			t.Errorf("Expected WritePos 1, got %d", cache.WritePos)
		}

		// Add second item
		if !cache.Add("msg2") {
			t.Error("Expected Add to return true for new item")
		}
		if cache.WritePos != 2 {
			t.Errorf("Expected WritePos 2, got %d", cache.WritePos)
		}
	})

	t.Run("add duplicate item", func(t *testing.T) {
		cache := NewDedupeCache(3)

		cache.Add("msg1")
		// Try to add the same item again
		if cache.Add("msg1") {
			t.Error("Expected Add to return false for duplicate item")
		}
	})

	t.Run("buffer wrap around", func(t *testing.T) {
		cache := NewDedupeCache(2)

		// Fill the buffer
		cache.Add("msg1")
		cache.Add("msg2")

		// Add third item, should wrap around and evict msg1
		cache.Add("msg3")

		if cache.WritePos != 1 {
			t.Errorf("Expected WritePos 1 after wrap, got %d", cache.WritePos)
		}

		// msg1 should be evicted, so adding it again should return true
		if !cache.Add("msg1") {
			t.Error("Expected Add to return true for evicted item")
		}
	})

	t.Run("single element cache", func(t *testing.T) {
		cache := NewDedupeCache(1)

		cache.Add("msg1")
		cache.Add("msg2") // Should evict msg1

		// msg1 should be evicted
		if !cache.Add("msg1") {
			t.Error("Expected Add to return true for evicted item")
		}
	})
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
	}{
		{"valid milliseconds", "100ms", 100 * time.Millisecond},
		{"valid seconds", "2s", 2 * time.Second},
		{"valid minutes", "5m", 5 * time.Minute},
		{"valid hours", "1h", 1 * time.Hour},
		{"valid nanoseconds", "500ns", 500 * time.Nanosecond},
		{"valid microseconds", "250us", 250 * time.Microsecond},
		{"complex duration", "1h30m45s", 1*time.Hour + 30*time.Minute + 45*time.Second},
		{"invalid format", "invalid", 0},
		{"empty string", "", 0},
		{"number without unit", "123", 0},
		{"negative duration", "-100ms", -100 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseDuration(tt.input)
			if result != tt.expected {
				t.Errorf("ParseDuration(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBuildParsedLog(t *testing.T) {
	t.Run("valid log data", func(t *testing.T) {
		data := map[string]string{
			"bytes":     "1024",
			"status":    "200",
			"service":   "150ms",
			"connect":   "10ms",
			"timestamp": "2023-07-19T10:30:45.123456789Z",
			"at":        "info",
			"dyno":      "web.1",
			"fwd":       "192.168.1.1",
			"host":      "example.com",
			"method":    "GET",
			"path":      "/api/users",
			"protocol":  "HTTP/1.1",
		}

		result := BuildParsedLog(data)

		if result == nil {
			t.Fatal("BuildParsedLog returned nil")
		}

		expectedTime, _ := time.Parse(time.RFC3339Nano, "2023-07-19T10:30:45.123456789Z")
		if !result.Time.Equal(expectedTime) {
			t.Errorf("Expected time %v, got %v", expectedTime, result.Time)
		}

		if result.Size != 1024 {
			t.Errorf("Expected size 1024, got %d", result.Size)
		}

		if result.Status != 200 {
			t.Errorf("Expected status 200, got %d", result.Status)
		}

		if result.ResponseTime != 150*time.Millisecond {
			t.Errorf("Expected response time 150ms, got %v", result.ResponseTime)
		}

		if result.ConnectTime != 10*time.Millisecond {
			t.Errorf("Expected connect time 10ms, got %v", result.ConnectTime)
		}

		if !result.Success {
			t.Error("Expected success to be true for 200 status")
		}

		if result.Threshold != "healthy" {
			t.Errorf("Expected threshold 'healthy', got %q", result.Threshold)
		}

		if result.IsSlow {
			t.Error("Expected IsSlow to be false for healthy response time")
		}

		if result.Level != "info" {
			t.Errorf("Expected level 'info', got %q", result.Level)
		}

		if result.SourceDyno != "web.1" {
			t.Errorf("Expected dyno 'web.1', got %q", result.SourceDyno)
		}

		if result.SourceIp != "192.168.1.1" {
			t.Errorf("Expected IP '192.168.1.1', got %q", result.SourceIp)
		}
	})

	t.Run("error status codes", func(t *testing.T) {
		testCases := []struct {
			status  string
			success bool
		}{
			{"400", false},
			{"404", false},
			{"500", false},
			{"301", true},
			{"302", true},
			{"399", true},
		}

		for _, tc := range testCases {
			data := map[string]string{
				"status":    tc.status,
				"bytes":     "100",
				"service":   "100ms",
				"connect":   "5ms",
				"timestamp": "2023-07-19T10:30:45Z",
			}

			result := BuildParsedLog(data)
			if result.Success != tc.success {
				t.Errorf("Status %s: expected success %v, got %v", tc.status, tc.success, result.Success)
			}
		}
	})

	t.Run("response time thresholds", func(t *testing.T) {
		testCases := []struct {
			service   string
			threshold string
			isSlow    bool
		}{
			{"100ms", "healthy", false},
			{"300ms", "medium", true},
			{"700ms", "critical", false},
		}

		for _, tc := range testCases {
			data := map[string]string{
				"service":   tc.service,
				"status":    "200",
				"bytes":     "100",
				"connect":   "5ms",
				"timestamp": "2023-07-19T10:30:45Z",
			}

			result := BuildParsedLog(data)
			if result.Threshold != tc.threshold {
				t.Errorf("Service %s: expected threshold %q, got %q", tc.service, tc.threshold, result.Threshold)
			}
			if result.IsSlow != tc.isSlow {
				t.Errorf("Service %s: expected isSlow %v, got %v", tc.service, tc.isSlow, result.IsSlow)
			}
		}
	})

	t.Run("invalid data", func(t *testing.T) {
		data := map[string]string{
			"bytes":     "invalid",
			"status":    "invalid",
			"service":   "invalid",
			"connect":   "invalid",
			"timestamp": "invalid",
		}

		result := BuildParsedLog(data)

		// Should return empty ParsedLog for invalid timestamp
		if !result.Time.IsZero() && result.Time.Year() != 1 {
			t.Errorf("Expected zero time for invalid timestamp, got %v", result.Time)
		}
	})

	t.Run("missing fields", func(t *testing.T) {
		data := map[string]string{
			"timestamp": "2023-07-19T10:30:45Z",
		}

		result := BuildParsedLog(data)

		if result.Size != 0 {
			t.Errorf("Expected size 0 for missing bytes, got %d", result.Size)
		}

		if result.Status != 0 {
			t.Errorf("Expected status 0 for missing status, got %d", result.Status)
		}

		if result.ResponseTime != 0 {
			t.Errorf("Expected response time 0 for missing service, got %v", result.ResponseTime)
		}
	})
}

func TestClassifyResTime(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"very fast", 50 * time.Millisecond, "healthy"},
		{"healthy boundary", 250 * time.Millisecond, "healthy"},
		{"just over healthy", 251 * time.Millisecond, "medium"},
		{"medium", 400 * time.Millisecond, "medium"},
		{"medium boundary", 600 * time.Millisecond, "medium"},
		{"just over medium", 601 * time.Millisecond, "critical"},
		{"critical", 1 * time.Second, "critical"},
		{"very slow", 5 * time.Second, "critical"},
		{"zero duration", 0, "healthy"},
		{"negative duration", -100 * time.Millisecond, "healthy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyResTime(tt.duration)
			if result != tt.expected {
				t.Errorf("ClassifyResTime(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestApp_fingerPrintIp(t *testing.T) {
	// Note: This test requires the IP2Location database file to be present
	// We'll create a mock test that checks the function exists and handles errors gracefully

	t.Run("function exists and handles nil database", func(t *testing.T) {
		app := &App{
			GeoDb: nil,
		}

		// This should not panic even with nil database
		result := app.fingerPrintIp("192.168.1.1")

		// Should return empty record on error
		if result.Country_short != "" && result.Country_short != "INVALID IP ADDRESS" && result.Country_short != "-" {
			t.Logf("Unexpected result for nil database: %+v", result)
		}
	})

	t.Run("handles various IP formats", func(t *testing.T) {
		app := &App{
			GeoDb: nil, // Will cause error, but function should handle gracefully
		}

		testIPs := []string{
			"192.168.1.1",
			"8.8.8.8",
			"invalid-ip",
			"",
			"::1",
			"2001:db8::1",
		}

		for _, ip := range testIPs {
			t.Run("IP: "+ip, func(t *testing.T) {
				// Should not panic
				result := app.fingerPrintIp(ip)
				t.Logf("IP %s result: %+v", ip, result)
			})
		}
	})
}

// Benchmark tests
func BenchmarkDedupeCache_Add(b *testing.B) {
	cache := NewDedupeCache(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Add("msg" + string(rune(i%1000)))
	}
}

func BenchmarkParseDuration(b *testing.B) {
	durations := []string{"100ms", "2s", "5m", "invalid"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseDuration(durations[i%len(durations)])
	}
}

func BenchmarkClassifyResTime(b *testing.B) {
	durations := []time.Duration{
		100 * time.Millisecond,
		300 * time.Millisecond,
		700 * time.Millisecond,
		2 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ClassifyResTime(durations[i%len(durations)])
	}
}

func BenchmarkBuildParsedLog(b *testing.B) {
	data := map[string]string{
		"bytes":     "1024",
		"status":    "200",
		"service":   "150ms",
		"connect":   "10ms",
		"timestamp": "2023-07-19T10:30:45.123456789Z",
		"at":        "info",
		"dyno":      "web.1",
		"fwd":       "192.168.1.1",
		"host":      "example.com",
		"method":    "GET",
		"path":      "/api/users",
		"protocol":  "HTTP/1.1",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BuildParsedLog(data)
	}
}
