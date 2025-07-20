package internal

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestApp_LogReceiver(t *testing.T) {
	createTestApp := func() *App {
		return &App{
			Dc:            NewDedupeCache(100),
			RawLogChan:    make(chan []byte, 10),
			ParsedLogChan: make(chan *ParsedLog, 10),
		}
	}

	t.Run("valid logplex request", func(t *testing.T) {
		app := createTestApp()

		body := "2023-07-19T10:30:45.123456+00:00 heroku[router]: at=info method=GET path=\"/api\" host=myapp.herokuapp.com request_id=abc123 fwd=\"192.168.1.1\" dyno=web.1 connect=1ms service=123ms status=200 bytes=456 protocol=https"
		req := httptest.NewRequest(http.MethodPost, "/logs", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/logplex-1")
		req.Header.Set("User-Agent", "Logplex/v73")
		req.Header.Set("Logplex-Msg-Count", "1")
		req.Header.Set("Logplex-Frame-Id", "frame-123")

		w := httptest.NewRecorder()

		app.LogReceiver(w, req)

		if w.Code != 0 && w.Code != http.StatusOK {
			t.Errorf("Expected no error status, got %d", w.Code)
		}

		select {
		case receivedLog := <-app.RawLogChan:
			if string(receivedLog) != body {
				t.Errorf("Expected log body %q, got %q", body, string(receivedLog))
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Expected log to be sent to RawLogChan")
		}
	})

	t.Run("invalid content type", func(t *testing.T) {
		app := createTestApp()

		req := httptest.NewRequest(http.MethodPost, "/logs", strings.NewReader("test log"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Logplex/v73")
		req.Header.Set("Logplex-Msg-Count", "1")
		req.Header.Set("Logplex-Frame-Id", "frame-123")

		w := httptest.NewRecorder()

		app.LogReceiver(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status %d, got %d", http.StatusNoContent, w.Code)
		}

		if w.Header().Get("Content-Length") != "0" {
			t.Errorf("Expected Content-Length 0, got %q", w.Header().Get("Content-Length"))
		}

		select {
		case <-app.RawLogChan:
			t.Error("Expected no log to be sent to RawLogChan")
		case <-time.After(50 * time.Millisecond):
		}
	})

	t.Run("invalid HTTP method", func(t *testing.T) {
		app := createTestApp()

		req := httptest.NewRequest(http.MethodGet, "/logs", nil)
		req.Header.Set("Content-Type", "application/logplex-1")
		req.Header.Set("User-Agent", "Logplex/v73")
		req.Header.Set("Logplex-Msg-Count", "1")
		req.Header.Set("Logplex-Frame-Id", "frame-123")

		w := httptest.NewRecorder()

		app.LogReceiver(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status %d, got %d", http.StatusNoContent, w.Code)
		}

		select {
		case <-app.RawLogChan:
			t.Error("Expected no log to be sent to RawLogChan")
		case <-time.After(50 * time.Millisecond):
		}
	})

	t.Run("invalid user agent", func(t *testing.T) {
		app := createTestApp()

		req := httptest.NewRequest(http.MethodPost, "/logs", strings.NewReader("test log"))
		req.Header.Set("Content-Type", "application/logplex-1")
		req.Header.Set("User-Agent", "Mozilla/5.0")
		req.Header.Set("Logplex-Msg-Count", "1")
		req.Header.Set("Logplex-Frame-Id", "frame-123")

		w := httptest.NewRecorder()

		app.LogReceiver(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status %d, got %d", http.StatusNoContent, w.Code)
		}

		if w.Header().Get("Content-Lenght") != "0" {
			t.Errorf("Expected Content-Lenght 0, got %q", w.Header().Get("Content-Lenght"))
		}

		select {
		case <-app.RawLogChan:
			t.Error("Expected no log to be sent to RawLogChan")
		case <-time.After(50 * time.Millisecond):
		}
	})

	t.Run("missing user agent", func(t *testing.T) {
		app := createTestApp()

		req := httptest.NewRequest(http.MethodPost, "/logs", strings.NewReader("test log"))
		req.Header.Set("Content-Type", "application/logplex-1")
		req.Header.Set("Logplex-Msg-Count", "1")
		req.Header.Set("Logplex-Frame-Id", "frame-123")

		w := httptest.NewRecorder()

		app.LogReceiver(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status %d, got %d", http.StatusNoContent, w.Code)
		}
	})

	t.Run("invalid message count - non-numeric", func(t *testing.T) {
		app := createTestApp()

		req := httptest.NewRequest(http.MethodPost, "/logs", strings.NewReader("test log"))
		req.Header.Set("Content-Type", "application/logplex-1")
		req.Header.Set("User-Agent", "Logplex/v73")
		req.Header.Set("Logplex-Msg-Count", "invalid")
		req.Header.Set("Logplex-Frame-Id", "frame-123")

		w := httptest.NewRecorder()

		app.LogReceiver(w, req)

		if w.Code != 0 && w.Code != http.StatusOK {
			t.Errorf("Expected no status code set, got %d", w.Code)
		}

		select {
		case <-app.RawLogChan:
			t.Error("Expected no log to be sent to RawLogChan")
		case <-time.After(50 * time.Millisecond):
		}
	})

	t.Run("invalid message count - zero", func(t *testing.T) {
		app := createTestApp()

		req := httptest.NewRequest(http.MethodPost, "/logs", strings.NewReader("test log"))
		req.Header.Set("Content-Type", "application/logplex-1")
		req.Header.Set("User-Agent", "Logplex/v73")
		req.Header.Set("Logplex-Msg-Count", "0")
		req.Header.Set("Logplex-Frame-Id", "frame-123")

		w := httptest.NewRecorder()

		app.LogReceiver(w, req)

		if w.Code != 0 && w.Code != http.StatusOK {
			t.Errorf("Expected no status code set, got %d", w.Code)
		}
	})

	t.Run("invalid message count - negative", func(t *testing.T) {
		app := createTestApp()

		req := httptest.NewRequest(http.MethodPost, "/logs", strings.NewReader("test log"))
		req.Header.Set("Content-Type", "application/logplex-1")
		req.Header.Set("User-Agent", "Logplex/v73")
		req.Header.Set("Logplex-Msg-Count", "-1")
		req.Header.Set("Logplex-Frame-Id", "frame-123")

		w := httptest.NewRecorder()

		app.LogReceiver(w, req)

		if w.Code != 0 && w.Code != http.StatusOK {
			t.Errorf("Expected no status code set, got %d", w.Code)
		}
	})

	t.Run("missing message count header", func(t *testing.T) {
		app := createTestApp()

		req := httptest.NewRequest(http.MethodPost, "/logs", strings.NewReader("test log"))
		req.Header.Set("Content-Type", "application/logplex-1")
		req.Header.Set("User-Agent", "Logplex/v73")
		req.Header.Set("Logplex-Frame-Id", "frame-123")

		w := httptest.NewRecorder()

		app.LogReceiver(w, req)

		if w.Code != 0 && w.Code != http.StatusOK {
			t.Errorf("Expected no status code set, got %d", w.Code)
		}
	})

	t.Run("duplicate request - same frame ID", func(t *testing.T) {
		app := createTestApp()

		body := "test log content"
		frameId := "duplicate-frame-123"

		// First request
		req1 := httptest.NewRequest(http.MethodPost, "/logs", strings.NewReader(body))
		req1.Header.Set("Content-Type", "application/logplex-1")
		req1.Header.Set("User-Agent", "Logplex/v73")
		req1.Header.Set("Logplex-Msg-Count", "1")
		req1.Header.Set("Logplex-Frame-Id", frameId)

		w1 := httptest.NewRecorder()
		app.LogReceiver(w1, req1)

		select {
		case receivedLog := <-app.RawLogChan:
			if string(receivedLog) != body {
				t.Errorf("Expected log body %q, got %q", body, string(receivedLog))
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Expected first log to be sent to RawLogChan")
		}

		// Second request with same frame ID
		req2 := httptest.NewRequest(http.MethodPost, "/logs", strings.NewReader(body))
		req2.Header.Set("Content-Type", "application/logplex-1")
		req2.Header.Set("User-Agent", "Logplex/v73")
		req2.Header.Set("Logplex-Msg-Count", "1")
		req2.Header.Set("Logplex-Frame-Id", frameId)

		w2 := httptest.NewRecorder()
		app.LogReceiver(w2, req2)

		select {
		case <-app.RawLogChan:
			t.Error("Expected no log to be sent for duplicate request")
		case <-time.After(50 * time.Millisecond):
		}
	})

	t.Run("multiple valid requests with different frame IDs", func(t *testing.T) {
		app := createTestApp()

		requests := []struct {
			body    string
			frameId string
		}{
			{"log message 1", "frame-1"},
			{"log message 2", "frame-2"},
			{"log message 3", "frame-3"},
		}

		for i, reqData := range requests {
			req := httptest.NewRequest(http.MethodPost, "/logs", strings.NewReader(reqData.body))
			req.Header.Set("Content-Type", "application/logplex-1")
			req.Header.Set("User-Agent", "Logplex/v73")
			req.Header.Set("Logplex-Msg-Count", "1")
			req.Header.Set("Logplex-Frame-Id", reqData.frameId)

			w := httptest.NewRecorder()
			app.LogReceiver(w, req)

			select {
			case receivedLog := <-app.RawLogChan:
				if string(receivedLog) != reqData.body {
					t.Errorf("Request %d: expected log body %q, got %q", i, reqData.body, string(receivedLog))
				}
			case <-time.After(100 * time.Millisecond):
				t.Errorf("Request %d: expected log to be sent to RawLogChan", i)
			}
		}
	})

	t.Run("large log body", func(t *testing.T) {
		app := createTestApp()

		// Create a large log message
		largeBody := strings.Repeat("This is a large log message. ", 1000)

		req := httptest.NewRequest(http.MethodPost, "/logs", strings.NewReader(largeBody))
		req.Header.Set("Content-Type", "application/logplex-1")
		req.Header.Set("User-Agent", "Logplex/v73")
		req.Header.Set("Logplex-Msg-Count", "1")
		req.Header.Set("Logplex-Frame-Id", "large-frame-123")

		w := httptest.NewRecorder()

		app.LogReceiver(w, req)

		select {
		case receivedLog := <-app.RawLogChan:
			if string(receivedLog) != largeBody {
				t.Error("Large log body was not received correctly")
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Expected large log to be sent to RawLogChan")
		}
	})

	t.Run("empty body", func(t *testing.T) {
		app := createTestApp()

		req := httptest.NewRequest(http.MethodPost, "/logs", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/logplex-1")
		req.Header.Set("User-Agent", "Logplex/v73")
		req.Header.Set("Logplex-Msg-Count", "1")
		req.Header.Set("Logplex-Frame-Id", "empty-frame-123")

		w := httptest.NewRecorder()

		app.LogReceiver(w, req)

		select {
		case receivedLog := <-app.RawLogChan:
			if len(receivedLog) != 0 {
				t.Errorf("Expected empty log body, got %d bytes", len(receivedLog))
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Expected empty log to be sent to RawLogChan")
		}
	})

	t.Run("different logplex versions", func(t *testing.T) {
		app := createTestApp()

		userAgents := []string{
			"Logplex/v73",
			"Logplex/v100",
			"Logplex/v1",
		}

		for _, ua := range userAgents {
			body := "test log " + ua
			req := httptest.NewRequest(http.MethodPost, "/logs", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/logplex-1")
			req.Header.Set("User-Agent", ua)
			req.Header.Set("Logplex-Msg-Count", "1")
			req.Header.Set("Logplex-Frame-Id", "frame-"+ua)

			w := httptest.NewRecorder()
			app.LogReceiver(w, req)

			select {
			case receivedLog := <-app.RawLogChan:
				if string(receivedLog) != body {
					t.Errorf("User-Agent %s: expected log body %q, got %q", ua, body, string(receivedLog))
				}
			case <-time.After(100 * time.Millisecond):
				t.Errorf("User-Agent %s: expected log to be sent to RawLogChan", ua)
			}
		}
	})

	t.Run("body read error simulation", func(t *testing.T) {
		app := createTestApp()

		// Create a request with a body that will cause a read error
		// We'll use a custom reader that returns an error
		errorReader := &errorReader{err: bytes.ErrTooLarge}
		req := httptest.NewRequest(http.MethodPost, "/logs", errorReader)
		req.Header.Set("Content-Type", "application/logplex-1")
		req.Header.Set("User-Agent", "Logplex/v73")
		req.Header.Set("Logplex-Msg-Count", "1")
		req.Header.Set("Logplex-Frame-Id", "error-frame-123")

		w := httptest.NewRecorder()

		app.LogReceiver(w, req)

		select {
		case <-app.RawLogChan:
			t.Error("Expected no log to be sent when body read fails")
		case <-time.After(50 * time.Millisecond):
		}
	})
}

// Helper type for simulating body read errors
type errorReader struct {
	err error
}

func (er *errorReader) Read(p []byte) (n int, err error) {
	return 0, er.err
}

func BenchmarkApp_LogReceiver(b *testing.B) {
	app := &App{
		Dc:            NewDedupeCache(b.N + 1000),
		RawLogChan:    make(chan []byte, 10000),
		ParsedLogChan: make(chan *ParsedLog, 10000),
	}

	body := "2023-07-19T10:30:45.123456+00:00 heroku[router]: at=info method=GET path=\"/api\" host=myapp.herokuapp.com request_id=abc123 fwd=\"192.168.1.1\" dyno=web.1 connect=1ms service=123ms status=200 bytes=456 protocol=https"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/logs", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/logplex-1")
		req.Header.Set("User-Agent", "Logplex/v73")
		req.Header.Set("Logplex-Msg-Count", "1")
		req.Header.Set("Logplex-Frame-Id", fmt.Sprintf("frame-%d", i)) // Unique frame ID for each iteration

		w := httptest.NewRecorder()
		app.LogReceiver(w, req)

		select {
		case <-app.RawLogChan:
		default:
		}
	}
}

func BenchmarkApp_LogReceiver_Duplicate(b *testing.B) {
	app := &App{
		Dc:            NewDedupeCache(10000),
		RawLogChan:    make(chan []byte, 10000),
		ParsedLogChan: make(chan *ParsedLog, 10000),
	}

	body := "test log for duplicate benchmark"
	frameId := "duplicate-frame"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/logs", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/logplex-1")
		req.Header.Set("User-Agent", "Logplex/v73")
		req.Header.Set("Logplex-Msg-Count", "1")
		req.Header.Set("Logplex-Frame-Id", frameId)

		w := httptest.NewRecorder()
		app.LogReceiver(w, req)

		select {
		case <-app.RawLogChan:
		default:
		}
	}
}
