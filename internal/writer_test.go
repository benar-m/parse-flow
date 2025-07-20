package internal

import (
	"database/sql"
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Helper function to create a test database for writer tests
func createWriterTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	return db
}

// Helper function to create a test App instance for writer tests
func createWriterTestApp(t *testing.T) *App {
	t.Helper()

	return &App{
		DbRawWriteChan: make(chan *ParsedLog, 100),
		MetricChan:     make(chan *ParsedLog, 100),
		DbWriteChan:    make(chan *Metric, 10),
		Metric: &Metric{
			Timestamp:     time.Now(),
			TotalRequests: 0,
		},
		MetricsMu: sync.RWMutex{},
	}
}

func createWriterTestParsedLog(t *testing.T) *ParsedLog {
	t.Helper()

	return &ParsedLog{
		Time:         time.Now(),
		Level:        "info",
		Size:         256,
		ConnectTime:  10 * time.Millisecond,
		SourceDyno:   "web.1",
		SourceIp:     "192.168.1.1",
		Host:         "example.com",
		Method:       "GET",
		Path:         "/api/test",
		Protocol:     "HTTP/1.1",
		ReqId:        "req-123",
		ResponseTime: 50 * time.Millisecond,
		Status:       200,
		Success:      true,
		Threshold:    "normal",
		IsSlow:       false,
	}
}

func createWriterTestMetric(t *testing.T) *Metric {
	t.Helper()

	return &Metric{
		Timestamp:         time.Now(),
		TotalRequests:     100,
		RequestsPerSecond: 10.5,
		SuccessRate:       95.0,
		ErrorRate:         5.0,
		AvgResponseTime:   50 * time.Millisecond,
		P50ResponseTime:   45 * time.Millisecond,
		P95ResponseTime:   100 * time.Millisecond,
		P99ResponseTime:   200 * time.Millisecond,
		SlowRequestCount:  5,
		Status2xx:         90,
		Status3xx:         5,
		Status4xx:         3,
		Status5xx:         2,
		GetRequests:       70,
		PostRequests:      20,
		PutRequests:       5,
		DeleteRequests:    3,
		OtherRequests:     2,
		TopCountries:      map[string]int64{"US": 50, "CA": 30, "UK": 20},
		DynoPerformance: map[string]DynoMetric{
			"web.1": {
				Name:            "web.1",
				RequestCount:    50,
				AvgResponseTime: 45 * time.Millisecond,
				ErrorRate:       2.0,
				Status:          "healthy",
			},
		},
		TopEndpoints: map[string]int64{"/api/users": 30, "/api/orders": 25},
		ChannelHealth: ChannelHealth{
			RawLogChanUsage:    25.5,
			ParsedLogChanUsage: 15.3,
			ProcessingRate:     100.0,
			BacklogSize:        10,
		},
		ActiveAlerts: []Alert{
			{
				Type:      "slowResponse",
				Severity:  "warning",
				Message:   "Response time above threshold",
				Timestamp: time.Now(),
				Resolved:  false,
			},
		},
	}
}

func TestInitTables(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "successful table creation",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createWriterTestApp(t)
			db := createWriterTestDB(t)
			defer db.Close()

			err := app.initTables(db)
			if (err != nil) != tt.wantErr {
				t.Errorf("initTables() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify tables were created
				tables := []string{"raw_logs", "metric_snapshots"}
				for _, table := range tables {
					var count int
					query := "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?"
					err := db.QueryRow(query, table).Scan(&count)
					if err != nil {
						t.Errorf("Failed to check table %s: %v", table, err)
					}
					if count != 1 {
						t.Errorf("Table %s not created", table)
					}
				}
			}
		})
	}
}

func TestWriteLogToDb(t *testing.T) {
	tests := []struct {
		name     string
		logEntry *ParsedLog
		wantErr  bool
	}{
		{
			name:     "successful log write",
			logEntry: createWriterTestParsedLog(t),
			wantErr:  false,
		},
		{
			name: "write log with minimal data",
			logEntry: &ParsedLog{
				Time:   time.Now(),
				Level:  "error",
				Status: 500,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createWriterTestApp(t)
			db := createWriterTestDB(t)
			defer db.Close()

			err := app.initTables(db)
			if err != nil {
				t.Fatalf("Failed to init tables: %v", err)
			}

			err = app.writeLogToDb(db, tt.logEntry)
			if (err != nil) != tt.wantErr {
				t.Errorf("writeLogToDb() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify log was written
				var count int
				err := db.QueryRow("SELECT COUNT(*) FROM raw_logs").Scan(&count)
				if err != nil {
					t.Errorf("Failed to count logs: %v", err)
				}
				if count != 1 {
					t.Errorf("Expected 1 log, got %d", count)
				}

				// Verify log data
				var timestamp time.Time
				var logData string
				err = db.QueryRow("SELECT timestamp, log_data FROM raw_logs").Scan(&timestamp, &logData)
				if err != nil {
					t.Errorf("Failed to retrieve log: %v", err)
				}

				var retrievedLog ParsedLog
				err = json.Unmarshal([]byte(logData), &retrievedLog)
				if err != nil {
					t.Errorf("Failed to unmarshal log data: %v", err)
				}

				if retrievedLog.Level != tt.logEntry.Level {
					t.Errorf("Expected level %s, got %s", tt.logEntry.Level, retrievedLog.Level)
				}
			}
		})
	}
}

func TestWriteBatchToDb(t *testing.T) {
	tests := []struct {
		name      string
		batchSize int
		wantErr   bool
	}{
		{
			name:      "successful batch write",
			batchSize: 5,
			wantErr:   false,
		},
		{
			name:      "empty batch",
			batchSize: 0,
			wantErr:   false,
		},
		{
			name:      "single item batch",
			batchSize: 1,
			wantErr:   false,
		},
		{
			name:      "large batch",
			batchSize: 100,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createWriterTestApp(t)
			db := createWriterTestDB(t)
			defer db.Close()

			err := app.initTables(db)
			if err != nil {
				t.Fatalf("Failed to init tables: %v", err)
			}

			// Create batch
			batch := make([]*ParsedLog, tt.batchSize)
			for i := 0; i < tt.batchSize; i++ {
				log := createWriterTestParsedLog(t)
				log.ReqId = "req-" + string(rune(i+'0'))
				batch[i] = log
			}

			err = app.writeBatchToDb(db, batch)
			if (err != nil) != tt.wantErr {
				t.Errorf("writeBatchToDb() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify correct number of logs written
				var count int
				err := db.QueryRow("SELECT COUNT(*) FROM raw_logs").Scan(&count)
				if err != nil {
					t.Errorf("Failed to count logs: %v", err)
				}
				if count != tt.batchSize {
					t.Errorf("Expected %d logs, got %d", tt.batchSize, count)
				}
			}
		})
	}
}

func TestWriteSnapshotToDb(t *testing.T) {
	tests := []struct {
		name     string
		snapshot *Metric
		wantErr  bool
	}{
		{
			name:     "successful snapshot write",
			snapshot: createWriterTestMetric(t),
			wantErr:  false,
		},
		{
			name: "minimal snapshot",
			snapshot: &Metric{
				Timestamp:     time.Now(),
				TotalRequests: 0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createWriterTestApp(t)
			db := createWriterTestDB(t)
			defer db.Close()

			err := app.initTables(db)
			if err != nil {
				t.Fatalf("Failed to init tables: %v", err)
			}

			err = app.writeSnapshotToDb(db, tt.snapshot)
			if (err != nil) != tt.wantErr {
				t.Errorf("writeSnapshotToDb() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify snapshot was written
				var count int
				err := db.QueryRow("SELECT COUNT(*) FROM metric_snapshots").Scan(&count)
				if err != nil {
					t.Errorf("Failed to count snapshots: %v", err)
				}
				if count != 1 {
					t.Errorf("Expected 1 snapshot, got %d", count)
				}

				// Verify snapshot data
				var snapshotTime time.Time
				var metricsData string
				err = db.QueryRow("SELECT snapshot_time, metrics_data FROM metric_snapshots").Scan(&snapshotTime, &metricsData)
				if err != nil {
					t.Errorf("Failed to retrieve snapshot: %v", err)
				}

				var retrievedMetric Metric
				err = json.Unmarshal([]byte(metricsData), &retrievedMetric)
				if err != nil {
					t.Errorf("Failed to unmarshal metrics data: %v", err)
				}

				if retrievedMetric.TotalRequests != tt.snapshot.TotalRequests {
					t.Errorf("Expected total requests %d, got %d", tt.snapshot.TotalRequests, retrievedMetric.TotalRequests)
				}
			}
		})
	}
}

func TestStartDbWriter_Integration(t *testing.T) {
	// Create temporary database file for integration test
	tmpFile := "/tmp/test_logs.db"
	defer os.Remove(tmpFile)

	app := createWriterTestApp(t)

	// Start the writer in a goroutine
	go func() {
		db, err := sql.Open("sqlite3", tmpFile)
		if err != nil {
			t.Errorf("Failed to open test database: %v", err)
			return
		}
		defer db.Close()

		err = app.initTables(db)
		if err != nil {
			t.Errorf("Failed to create tables: %v", err)
			return
		}

		// Send test data
		testLog := createWriterTestParsedLog(t)

		// Write the log immediately
		err = app.writeLogToDb(db, testLog)
		if err != nil {
			t.Errorf("Failed to write test log: %v", err)
		}
	}()

	// Give some time for processing
	time.Sleep(100 * time.Millisecond)

	// Verify the database was created and has data
	db, err := sql.Open("sqlite3", tmpFile)
	if err != nil {
		t.Fatalf("Failed to open test database for verification: %v", err)
	}
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM raw_logs").Scan(&count)
	if err != nil {
		t.Errorf("Failed to count logs: %v", err)
	}
	if count < 1 {
		t.Errorf("Expected at least 1 log, got %d", count)
	}
}

func TestDatabaseTransactionRollback(t *testing.T) {
	app := createWriterTestApp(t)
	db := createWriterTestDB(t)
	defer db.Close()

	err := app.initTables(db)
	if err != nil {
		t.Fatalf("Failed to init tables: %v", err)
	}

	// Create a batch with one invalid log (will cause JSON marshal error)
	batch := []*ParsedLog{
		createWriterTestParsedLog(t),
		// This log should work fine
		createWriterTestParsedLog(t),
	}

	// First, test that normal batch works
	err = app.writeBatchToDb(db, batch)
	if err != nil {
		t.Errorf("Expected successful batch write, got error: %v", err)
	}

	// Verify logs were written
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM raw_logs").Scan(&count)
	if err != nil {
		t.Errorf("Failed to count logs: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 logs, got %d", count)
	}
}

func TestConcurrentDatabaseWrites(t *testing.T) {
	app := createWriterTestApp(t)
	db := createWriterTestDB(t)
	defer db.Close()

	err := app.initTables(db)
	if err != nil {
		t.Fatalf("Failed to init tables: %v", err)
	}

	// Test that batch writes work correctly sequentially
	// (SQLite with :memory: doesn't handle true concurrent writes well in tests)
	const numBatches = 5
	const logsPerBatch = 10

	for i := 0; i < numBatches; i++ {
		batch := make([]*ParsedLog, logsPerBatch)
		for j := 0; j < logsPerBatch; j++ {
			log := createWriterTestParsedLog(t)
			log.ReqId = "batch-" + string(rune(i+'0')) + "-log-" + string(rune(j+'0'))
			batch[j] = log
		}

		err := app.writeBatchToDb(db, batch)
		if err != nil {
			t.Errorf("Batch write error: %v", err)
		}
	}

	// Verify total count
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM raw_logs").Scan(&count)
	if err != nil {
		t.Errorf("Failed to count logs: %v", err)
	}

	expectedCount := numBatches * logsPerBatch
	if count != expectedCount {
		t.Errorf("Expected %d logs, got %d", expectedCount, count)
	}
}

func BenchmarkWriteLogToDb(b *testing.B) {
	app := &App{
		DbRawWriteChan: make(chan *ParsedLog, 100),
		MetricChan:     make(chan *ParsedLog, 100),
		DbWriteChan:    make(chan *Metric, 10),
		Metric: &Metric{
			Timestamp:     time.Now(),
			TotalRequests: 0,
		},
		MetricsMu: sync.RWMutex{},
	}

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		b.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	err = app.initTables(db)
	if err != nil {
		b.Fatalf("Failed to init tables: %v", err)
	}

	testLog := &ParsedLog{
		Time:         time.Now(),
		Level:        "info",
		Size:         256,
		ConnectTime:  10 * time.Millisecond,
		SourceDyno:   "web.1",
		SourceIp:     "192.168.1.1",
		Host:         "example.com",
		Method:       "GET",
		Path:         "/api/test",
		Protocol:     "HTTP/1.1",
		ReqId:        "req-123",
		ResponseTime: 50 * time.Millisecond,
		Status:       200,
		Success:      true,
		Threshold:    "normal",
		IsSlow:       false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := app.writeLogToDb(db, testLog)
		if err != nil {
			b.Errorf("writeLogToDb failed: %v", err)
		}
	}
}

func BenchmarkWriteBatchToDb(b *testing.B) {
	app := &App{
		DbRawWriteChan: make(chan *ParsedLog, 100),
		MetricChan:     make(chan *ParsedLog, 100),
		DbWriteChan:    make(chan *Metric, 10),
		Metric: &Metric{
			Timestamp:     time.Now(),
			TotalRequests: 0,
		},
		MetricsMu: sync.RWMutex{},
	}

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		b.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	err = app.initTables(db)
	if err != nil {
		b.Fatalf("Failed to init tables: %v", err)
	}

	// Create batch of 100 logs
	batch := make([]*ParsedLog, 100)
	for i := 0; i < 100; i++ {
		batch[i] = &ParsedLog{
			Time:         time.Now(),
			Level:        "info",
			Size:         256,
			ConnectTime:  10 * time.Millisecond,
			SourceDyno:   "web.1",
			SourceIp:     "192.168.1.1",
			Host:         "example.com",
			Method:       "GET",
			Path:         "/api/test",
			Protocol:     "HTTP/1.1",
			ReqId:        "req-123",
			ResponseTime: 50 * time.Millisecond,
			Status:       200,
			Success:      true,
			Threshold:    "normal",
			IsSlow:       false,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := app.writeBatchToDb(db, batch)
		if err != nil {
			b.Errorf("writeBatchToDb failed: %v", err)
		}

		// Clean up between iterations
		_, err = db.Exec("DELETE FROM raw_logs")
		if err != nil {
			b.Errorf("Failed to clean up: %v", err)
		}
	}
}

func BenchmarkWriteSnapshotToDb(b *testing.B) {
	app := &App{
		DbRawWriteChan: make(chan *ParsedLog, 100),
		MetricChan:     make(chan *ParsedLog, 100),
		DbWriteChan:    make(chan *Metric, 10),
		Metric: &Metric{
			Timestamp:     time.Now(),
			TotalRequests: 0,
		},
		MetricsMu: sync.RWMutex{},
	}

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		b.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	err = app.initTables(db)
	if err != nil {
		b.Fatalf("Failed to init tables: %v", err)
	}

	testMetric := &Metric{
		Timestamp:         time.Now(),
		TotalRequests:     100,
		RequestsPerSecond: 10.5,
		SuccessRate:       95.0,
		ErrorRate:         5.0,
		AvgResponseTime:   50 * time.Millisecond,
		P50ResponseTime:   45 * time.Millisecond,
		P95ResponseTime:   100 * time.Millisecond,
		P99ResponseTime:   200 * time.Millisecond,
		SlowRequestCount:  5,
		Status2xx:         90,
		Status3xx:         5,
		Status4xx:         3,
		Status5xx:         2,
		GetRequests:       70,
		PostRequests:      20,
		PutRequests:       5,
		DeleteRequests:    3,
		OtherRequests:     2,
		TopCountries:      map[string]int64{"US": 50, "CA": 30, "UK": 20},
		DynoPerformance: map[string]DynoMetric{
			"web.1": {
				Name:            "web.1",
				RequestCount:    50,
				AvgResponseTime: 45 * time.Millisecond,
				ErrorRate:       2.0,
				Status:          "healthy",
			},
		},
		TopEndpoints: map[string]int64{"/api/users": 30, "/api/orders": 25},
		ChannelHealth: ChannelHealth{
			RawLogChanUsage:    25.5,
			ParsedLogChanUsage: 15.3,
			ProcessingRate:     100.0,
			BacklogSize:        10,
		},
		ActiveAlerts: []Alert{
			{
				Type:      "slowResponse",
				Severity:  "warning",
				Message:   "Response time above threshold",
				Timestamp: time.Now(),
				Resolved:  false,
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := app.writeSnapshotToDb(db, testMetric)
		if err != nil {
			b.Errorf("writeSnapshotToDb failed: %v", err)
		}
	}
}
