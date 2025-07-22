package internal

import (
	"sync"
	"testing"
	"time"

	ip2 "github.com/ip2location/ip2location-go"
)

func createTestAppForMetrics() *App {
	return &App{
		Dc:             NewDedupeCache(100),
		RawLogChan:     make(chan []byte, 100),
		ParsedLogChan:  make(chan *ParsedLog, 100),
		MetricChan:     make(chan *ParsedLog, 100),
		DbWriteChan:    make(chan *Metric, 100),
		DbRawWriteChan: make(chan *ParsedLog, 100),
		MetricsMu:      sync.RWMutex{},
	}
}

func createTestParsedLog(status int, method string, path string, sourceIp string, sourceDyno string, responseTime time.Duration, isSlow bool) *ParsedLog {
	return &ParsedLog{
		Time:         time.Now(),
		Level:        "info",
		Size:         1024,
		ConnectTime:  10 * time.Millisecond,
		SourceDyno:   sourceDyno,
		SourceIp:     sourceIp,
		Host:         "example.com",
		Method:       method,
		Path:         path,
		Protocol:     "https",
		ReqId:        "req-123",
		ResponseTime: responseTime,
		Status:       status,
		Success:      status >= 200 && status < 300,
		Threshold:    "normal",
		IsSlow:       isSlow,
	}
}

func TestApp_StartMetricsAggregator(t *testing.T) {
	t.Run("initialization", func(t *testing.T) {
		app := createTestAppForMetrics()

		app.StartMetricsAggregator()

		if app.Metric == nil {
			t.Error("Expected Metric to be initialized")
		}

		metrics := app.GetMetricsSnapshot()

		if metrics.TopCountries == nil {
			t.Error("Expected TopCountries to be initialized")
		}

		if metrics.DynoPerformance == nil {
			t.Error("Expected DynoPerformance to be initialized")
		}

		if metrics.TopEndpoints == nil {
			t.Error("Expected TopEndpoints to be initialized")
		}

		if metrics.ActiveAlerts == nil {
			t.Error("Expected ActiveAlerts to be initialized")
		}
	})

	t.Run("status code classification", func(t *testing.T) {
		app := createTestAppForMetrics()
		app.StartMetricsAggregator()

		testCases := []struct {
			status   int
			expected string
		}{
			{200, "2xx"},
			{201, "2xx"},
			{299, "2xx"},
			{300, "3xx"},
			{301, "3xx"},
			{399, "3xx"},
			{400, "4xx"},
			{404, "4xx"},
			{499, "4xx"},
			{500, "5xx"},
			{503, "5xx"},
		}

		for _, tc := range testCases {
			log := createTestParsedLog(tc.status, "GET", "/test", "", "", 100*time.Millisecond, false)
			app.MetricChan <- log
		}

		time.Sleep(100 * time.Millisecond)

		metrics := app.GetMetricsSnapshot()

		if metrics.Status2xx != 3 {
			t.Errorf("Expected 3 2xx responses, got %d", metrics.Status2xx)
		}
		if metrics.Status3xx != 3 {
			t.Errorf("Expected 3 3xx responses, got %d", metrics.Status3xx)
		}
		if metrics.Status4xx != 3 {
			t.Errorf("Expected 3 4xx responses, got %d", metrics.Status4xx)
		}
		if metrics.Status5xx != 2 {
			t.Errorf("Expected 2 5xx responses, got %d", metrics.Status5xx)
		}
		if metrics.TotalRequests != 11 {
			t.Errorf("Expected 11 total requests, got %d", metrics.TotalRequests)
		}
	})

	t.Run("HTTP method classification", func(t *testing.T) {
		app := createTestAppForMetrics()
		app.StartMetricsAggregator()

		methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
		for _, method := range methods {
			log := createTestParsedLog(200, method, "/test", "", "", 100*time.Millisecond, false)
			app.MetricChan <- log
		}

		time.Sleep(100 * time.Millisecond)

		// Use GetMetricsSnapshot to safely read metrics
		metrics := app.GetMetricsSnapshot()

		if metrics.GetRequests != 1 {
			t.Errorf("Expected 1 GET request, got %d", metrics.GetRequests)
		}
		if metrics.PostRequests != 1 {
			t.Errorf("Expected 1 POST request, got %d", metrics.PostRequests)
		}
		if metrics.PutRequests != 1 {
			t.Errorf("Expected 1 PUT request, got %d", metrics.PutRequests)
		}
		if metrics.DeleteRequests != 1 {
			t.Errorf("Expected 1 DELETE request, got %d", metrics.DeleteRequests)
		}
		if metrics.OtherRequests != 1 {
			t.Errorf("Expected 1 other request, got %d", metrics.OtherRequests)
		}
	})

	t.Run("slow request tracking", func(t *testing.T) {
		app := createTestAppForMetrics()
		app.StartMetricsAggregator()

		log1 := createTestParsedLog(200, "GET", "/test", "", "", 100*time.Millisecond, false)
		log2 := createTestParsedLog(200, "GET", "/test", "", "", 2*time.Second, true)
		log3 := createTestParsedLog(200, "GET", "/test", "", "", 3*time.Second, true)

		app.MetricChan <- log1
		app.MetricChan <- log2
		app.MetricChan <- log3

		time.Sleep(100 * time.Millisecond)

		metrics := app.GetMetricsSnapshot()
		if metrics.SlowRequestCount != 2 {
			t.Errorf("Expected 2 slow requests, got %d", metrics.SlowRequestCount)
		}
	})

	t.Run("top endpoints tracking", func(t *testing.T) {
		app := createTestAppForMetrics()
		app.StartMetricsAggregator()

		paths := []string{"/api/users", "/api/posts", "/api/users", "/api/comments", "/api/users"}
		for _, path := range paths {
			log := createTestParsedLog(200, "GET", path, "", "", 100*time.Millisecond, false)
			app.MetricChan <- log
		}

		time.Sleep(100 * time.Millisecond)

		metrics := app.GetMetricsSnapshot()
		if metrics.TopEndpoints["/api/users"] != 3 {
			t.Errorf("Expected 3 requests to /api/users, got %d", metrics.TopEndpoints["/api/users"])
		}
		if metrics.TopEndpoints["/api/posts"] != 1 {
			t.Errorf("Expected 1 request to /api/posts, got %d", metrics.TopEndpoints["/api/posts"])
		}
	})

	t.Run("dyno performance tracking", func(t *testing.T) {
		app := createTestAppForMetrics()
		app.StartMetricsAggregator()

		log1 := createTestParsedLog(200, "GET", "/test", "", "web.1", 100*time.Millisecond, false)
		log2 := createTestParsedLog(500, "GET", "/test", "", "web.1", 200*time.Millisecond, false)
		log3 := createTestParsedLog(200, "GET", "/test", "", "web.2", 50*time.Millisecond, false)

		app.MetricChan <- log1
		app.MetricChan <- log2
		app.MetricChan <- log3

		time.Sleep(100 * time.Millisecond)

		dyno1 := app.Metric.DynoPerformance["web.1"]
		if dyno1.RequestCount != 2 {
			t.Errorf("Expected web.1 to have 2 requests, got %d", dyno1.RequestCount)
		}
		if dyno1.ErrorRate != 50.0 {
			t.Errorf("Expected web.1 error rate to be 50.0, got %f", dyno1.ErrorRate)
		}
		if dyno1.Status != "critical" {
			t.Errorf("Expected web.1 status to be critical, got %s", dyno1.Status)
		}

		dyno2 := app.Metric.DynoPerformance["web.2"]
		if dyno2.RequestCount != 1 {
			t.Errorf("Expected web.2 to have 1 request, got %d", dyno2.RequestCount)
		}
		if dyno2.Status != "healthy" {
			t.Errorf("Expected web.2 status to be healthy, got %s", dyno2.Status)
		}
	})

	t.Run("country tracking with geodb", func(t *testing.T) {
		app := createTestAppForMetrics()

		mockDb := &ip2.DB{}
		app.GeoDb = mockDb

		app.StartMetricsAggregator()

		log1 := createTestParsedLog(200, "GET", "/test", "192.168.1.1", "", 100*time.Millisecond, false)
		log2 := createTestParsedLog(200, "GET", "/test", `"10.0.0.1"`, "", 100*time.Millisecond, false)

		app.MetricChan <- log1
		app.MetricChan <- log2

		time.Sleep(100 * time.Millisecond)
	})

	t.Run("success and error rates calculation", func(t *testing.T) {
		app := createTestAppForMetrics()
		app.StartMetricsAggregator()

		logs := []*ParsedLog{
			createTestParsedLog(200, "GET", "/test", "", "", 100*time.Millisecond, false),
			createTestParsedLog(200, "GET", "/test", "", "", 100*time.Millisecond, false),
			createTestParsedLog(400, "GET", "/test", "", "", 100*time.Millisecond, false),
			createTestParsedLog(500, "GET", "/test", "", "", 100*time.Millisecond, false),
		}

		for _, log := range logs {
			app.MetricChan <- log
		}

		time.Sleep(100 * time.Millisecond)

		expectedSuccessRate := 50.0 // 2 out of 4 requests are 2xx
		expectedErrorRate := 50.0   // 2 out of 4 requests are 4xx/5xx

		if app.Metric.SuccessRate != expectedSuccessRate {
			t.Errorf("Expected success rate %f, got %f", expectedSuccessRate, app.Metric.SuccessRate)
		}
		if app.Metric.ErrorRate != expectedErrorRate {
			t.Errorf("Expected error rate %f, got %f", expectedErrorRate, app.Metric.ErrorRate)
		}
	})

	t.Run("requests per second calculation", func(t *testing.T) {
		app := createTestAppForMetrics()
		app.StartMetricsAggregator()

		log := createTestParsedLog(200, "GET", "/test", "", "", 100*time.Millisecond, false)
		app.MetricChan <- log

		time.Sleep(100 * time.Millisecond)

		if app.Metric.RequestsPerSecond <= 0 {
			t.Errorf("Expected positive requests per second, got %f", app.Metric.RequestsPerSecond)
		}
	})

	t.Run("average response time calculation", func(t *testing.T) {
		app := createTestAppForMetrics()
		app.StartMetricsAggregator()

		logs := []*ParsedLog{
			createTestParsedLog(200, "GET", "/test", "", "", 100*time.Millisecond, false),
			createTestParsedLog(200, "GET", "/test", "", "", 200*time.Millisecond, false),
			createTestParsedLog(200, "GET", "/test", "", "", 300*time.Millisecond, false),
		}

		for _, log := range logs {
			app.MetricChan <- log
		}

		time.Sleep(100 * time.Millisecond)

		expectedAvg := 200 * time.Millisecond
		if app.Metric.AvgResponseTime != expectedAvg {
			t.Errorf("Expected average response time %v, got %v", expectedAvg, app.Metric.AvgResponseTime)
		}
	})
}

func TestApp_calculatePercentiles(t *testing.T) {
	t.Run("empty response times", func(t *testing.T) {
		app := createTestAppForMetrics()
		app.Metric = &Metric{}

		aggregator := &MetricsAggregator{
			responseTimes: []time.Duration{},
		}

		app.calculatePercentiles(aggregator)

		if app.Metric.P50ResponseTime != 0 {
			t.Errorf("Expected P50 to be 0 for empty response times, got %v", app.Metric.P50ResponseTime)
		}
	})

	t.Run("valid percentile calculation", func(t *testing.T) {
		app := createTestAppForMetrics()
		app.Metric = &Metric{}

		responseTimes := make([]time.Duration, 100)
		for i := 0; i < 100; i++ {
			responseTimes[i] = time.Duration(i+1) * time.Millisecond
		}

		aggregator := &MetricsAggregator{
			responseTimes: responseTimes,
		}

		app.calculatePercentiles(aggregator)

		if app.Metric.P50ResponseTime != 51*time.Millisecond {
			t.Errorf("Expected P50 to be 51ms, got %v", app.Metric.P50ResponseTime)
		}
		if app.Metric.P95ResponseTime != 96*time.Millisecond {
			t.Errorf("Expected P95 to be 96ms, got %v", app.Metric.P95ResponseTime)
		}
		if app.Metric.P99ResponseTime != 100*time.Millisecond {
			t.Errorf("Expected P99 to be 100ms, got %v", app.Metric.P99ResponseTime)
		}
	})

	t.Run("response times pruning", func(t *testing.T) {
		app := createTestAppForMetrics()
		app.Metric = &Metric{}

		responseTimes := make([]time.Duration, 1500)
		for i := 0; i < 1500; i++ {
			responseTimes[i] = time.Duration(i+1) * time.Millisecond
		}

		aggregator := &MetricsAggregator{
			responseTimes: responseTimes,
		}

		app.calculatePercentiles(aggregator)

		if len(aggregator.responseTimes) != 500 {
			t.Errorf("Expected response times to be pruned to 500, got %d", len(aggregator.responseTimes))
		}
	})

	t.Run("small dataset percentiles", func(t *testing.T) {
		app := createTestAppForMetrics()
		app.Metric = &Metric{}

		responseTimes := []time.Duration{
			10 * time.Millisecond,
			20 * time.Millisecond,
			30 * time.Millisecond,
		}

		aggregator := &MetricsAggregator{
			responseTimes: responseTimes,
		}

		app.calculatePercentiles(aggregator)

		if app.Metric.P50ResponseTime != 20*time.Millisecond {
			t.Errorf("Expected P50 to be 20ms for small dataset, got %v", app.Metric.P50ResponseTime)
		}
	})
}

func TestApp_generateAlerts(t *testing.T) {
	t.Run("high error rate warning alert", func(t *testing.T) {
		app := createTestAppForMetrics()
		app.Metric = &Metric{
			ErrorRate:    7.5,
			ActiveAlerts: []Alert{},
		}

		app.generateAlerts()

		if len(app.Metric.ActiveAlerts) != 1 {
			t.Errorf("Expected 1 alert, got %d", len(app.Metric.ActiveAlerts))
		}

		alert := app.Metric.ActiveAlerts[0]
		if alert.Type != "high_error_rate" {
			t.Errorf("Expected high_error_rate alert, got %s", alert.Type)
		}
		if alert.Severity != "warning" {
			t.Errorf("Expected warning severity, got %s", alert.Severity)
		}
		if alert.Resolved {
			t.Error("Expected alert to not be resolved")
		}
	})

	t.Run("high error rate critical alert", func(t *testing.T) {
		app := createTestAppForMetrics()
		app.Metric = &Metric{
			ErrorRate:    15.0,
			ActiveAlerts: []Alert{},
		}

		app.generateAlerts()

		if len(app.Metric.ActiveAlerts) != 1 {
			t.Errorf("Expected 1 alert, got %d", len(app.Metric.ActiveAlerts))
		}

		alert := app.Metric.ActiveAlerts[0]
		if alert.Severity != "critical" {
			t.Errorf("Expected critical severity, got %s", alert.Severity)
		}
		if alert.Message != "Error rate is above 10%" {
			t.Errorf("Expected critical error rate message, got %s", alert.Message)
		}
	})

	t.Run("slow response warning alert", func(t *testing.T) {
		app := createTestAppForMetrics()
		app.Metric = &Metric{
			P95ResponseTime: 3 * time.Second,
			ActiveAlerts:    []Alert{},
		}

		app.generateAlerts()

		if len(app.Metric.ActiveAlerts) != 1 {
			t.Errorf("Expected 1 alert, got %d", len(app.Metric.ActiveAlerts))
		}

		alert := app.Metric.ActiveAlerts[0]
		if alert.Type != "slow_response" {
			t.Errorf("Expected slow_response alert, got %s", alert.Type)
		}
		if alert.Severity != "warning" {
			t.Errorf("Expected warning severity, got %s", alert.Severity)
		}
	})

	t.Run("slow response critical alert", func(t *testing.T) {
		app := createTestAppForMetrics()
		app.Metric = &Metric{
			P95ResponseTime: 6 * time.Second,
			ActiveAlerts:    []Alert{},
		}

		app.generateAlerts()

		alert := app.Metric.ActiveAlerts[0]
		if alert.Severity != "critical" {
			t.Errorf("Expected critical severity, got %s", alert.Severity)
		}
		if alert.Message != "P95 response time is above 5 seconds" {
			t.Errorf("Expected critical response time message, got %s", alert.Message)
		}
	})

	t.Run("multiple alerts", func(t *testing.T) {
		app := createTestAppForMetrics()
		app.Metric = &Metric{
			ErrorRate:       15.0,
			P95ResponseTime: 6 * time.Second,
			ActiveAlerts:    []Alert{},
		}

		app.generateAlerts()

		if len(app.Metric.ActiveAlerts) != 2 {
			t.Errorf("Expected 2 alerts, got %d", len(app.Metric.ActiveAlerts))
		}
	})

	t.Run("update existing alert", func(t *testing.T) {
		app := createTestAppForMetrics()
		existingAlert := Alert{
			Type:      "high_error_rate",
			Severity:  "warning",
			Message:   "Error rate is above 5%",
			Timestamp: time.Now().Add(-1 * time.Hour),
			Resolved:  false,
		}

		app.Metric = &Metric{
			ErrorRate:    15.0,
			ActiveAlerts: []Alert{existingAlert},
		}

		app.generateAlerts()

		if len(app.Metric.ActiveAlerts) != 1 {
			t.Errorf("Expected 1 alert after update, got %d", len(app.Metric.ActiveAlerts))
		}

		alert := app.Metric.ActiveAlerts[0]
		if alert.Severity != "critical" {
			t.Errorf("Expected alert to be updated to critical, got %s", alert.Severity)
		}
	})

	t.Run("alert cleanup", func(t *testing.T) {
		app := createTestAppForMetrics()
		oldAlert := Alert{
			Type:      "high_error_rate",
			Severity:  "warning",
			Message:   "Error rate is above 5%",
			Timestamp: time.Now().Add(-2 * time.Hour),
			Resolved:  true,
		}

		app.Metric = &Metric{
			ErrorRate:    2.0,
			ActiveAlerts: []Alert{oldAlert},
		}

		app.generateAlerts()

		if len(app.Metric.ActiveAlerts) != 0 {
			t.Errorf("Expected old resolved alerts to be cleaned up, got %d alerts", len(app.Metric.ActiveAlerts))
		}
	})

	t.Run("no alerts when metrics are healthy", func(t *testing.T) {
		app := createTestAppForMetrics()
		app.Metric = &Metric{
			ErrorRate:       2.0,
			P95ResponseTime: 500 * time.Millisecond,
			ActiveAlerts:    []Alert{},
		}

		app.generateAlerts()

		if len(app.Metric.ActiveAlerts) != 0 {
			t.Errorf("Expected no alerts for healthy metrics, got %d", len(app.Metric.ActiveAlerts))
		}
	})
}

func TestApp_updateChannelHealth(t *testing.T) {
	t.Run("channel usage calculation", func(t *testing.T) {
		app := createTestAppForMetrics()
		app.Metric = &Metric{}
		app.RawLogChan = make(chan []byte, 100)
		app.ParsedLogChan = make(chan *ParsedLog, 50)

		for i := 0; i < 25; i++ {
			app.RawLogChan <- []byte("test")
		}

		for i := 0; i < 10; i++ {
			app.ParsedLogChan <- &ParsedLog{}
		}

		app.Metric.RequestsPerSecond = 15.5

		app.updateChannelHealth()

		expectedRawUsage := 25.0
		expectedParsedUsage := 20.0
		expectedBacklogSize := 35

		if app.Metric.ChannelHealth.RawLogChanUsage != expectedRawUsage {
			t.Errorf("Expected raw channel usage %f, got %f", expectedRawUsage, app.Metric.ChannelHealth.RawLogChanUsage)
		}

		if app.Metric.ChannelHealth.ParsedLogChanUsage != expectedParsedUsage {
			t.Errorf("Expected parsed channel usage %f, got %f", expectedParsedUsage, app.Metric.ChannelHealth.ParsedLogChanUsage)
		}

		if app.Metric.ChannelHealth.ProcessingRate != 15.5 {
			t.Errorf("Expected processing rate 15.5, got %f", app.Metric.ChannelHealth.ProcessingRate)
		}

		if app.Metric.ChannelHealth.BacklogSize != expectedBacklogSize {
			t.Errorf("Expected backlog size %d, got %d", expectedBacklogSize, app.Metric.ChannelHealth.BacklogSize)
		}
	})

	t.Run("empty channels", func(t *testing.T) {
		app := createTestAppForMetrics()
		app.Metric = &Metric{}
		app.RawLogChan = make(chan []byte, 100)
		app.ParsedLogChan = make(chan *ParsedLog, 50)
		app.Metric.RequestsPerSecond = 0.0

		app.updateChannelHealth()

		if app.Metric.ChannelHealth.RawLogChanUsage != 0.0 {
			t.Errorf("Expected raw channel usage 0, got %f", app.Metric.ChannelHealth.RawLogChanUsage)
		}

		if app.Metric.ChannelHealth.ParsedLogChanUsage != 0.0 {
			t.Errorf("Expected parsed channel usage 0, got %f", app.Metric.ChannelHealth.ParsedLogChanUsage)
		}

		if app.Metric.ChannelHealth.BacklogSize != 0 {
			t.Errorf("Expected backlog size 0, got %d", app.Metric.ChannelHealth.BacklogSize)
		}
	})

	t.Run("full channels", func(t *testing.T) {
		app := createTestAppForMetrics()
		app.Metric = &Metric{}
		app.RawLogChan = make(chan []byte, 10)
		app.ParsedLogChan = make(chan *ParsedLog, 5)

		for i := 0; i < 10; i++ {
			app.RawLogChan <- []byte("test")
		}

		for i := 0; i < 5; i++ {
			app.ParsedLogChan <- &ParsedLog{}
		}

		app.updateChannelHealth()

		if app.Metric.ChannelHealth.RawLogChanUsage != 100.0 {
			t.Errorf("Expected raw channel usage 100, got %f", app.Metric.ChannelHealth.RawLogChanUsage)
		}

		if app.Metric.ChannelHealth.ParsedLogChanUsage != 100.0 {
			t.Errorf("Expected parsed channel usage 100, got %f", app.Metric.ChannelHealth.ParsedLogChanUsage)
		}

		if app.Metric.ChannelHealth.BacklogSize != 15 {
			t.Errorf("Expected backlog size 15, got %d", app.Metric.ChannelHealth.BacklogSize)
		}
	})
}

func TestMetricsAggregator_DynoHealthStatus(t *testing.T) {
	t.Run("healthy dyno status", func(t *testing.T) {
		app := createTestAppForMetrics()
		app.StartMetricsAggregator()

		log := createTestParsedLog(200, "GET", "/test", "", "web.1", 500*time.Millisecond, false)
		app.MetricChan <- log

		time.Sleep(100 * time.Millisecond)

		dyno := app.Metric.DynoPerformance["web.1"]
		if dyno.Status != "healthy" {
			t.Errorf("Expected healthy status, got %s", dyno.Status)
		}
	})

	t.Run("warning dyno status", func(t *testing.T) {
		app := createTestAppForMetrics()
		app.StartMetricsAggregator()

		log := createTestParsedLog(200, "GET", "/test", "", "web.1", 1500*time.Millisecond, false)
		app.MetricChan <- log

		time.Sleep(100 * time.Millisecond)

		dyno := app.Metric.DynoPerformance["web.1"]
		if dyno.Status != "warning" {
			t.Errorf("Expected warning status, got %s", dyno.Status)
		}
	})

	t.Run("critical dyno status due to response time", func(t *testing.T) {
		app := createTestAppForMetrics()
		app.StartMetricsAggregator()

		log := createTestParsedLog(200, "GET", "/test", "", "web.1", 3*time.Second, false)
		app.MetricChan <- log

		time.Sleep(100 * time.Millisecond)

		dyno := app.Metric.DynoPerformance["web.1"]
		if dyno.Status != "critical" {
			t.Errorf("Expected critical status, got %s", dyno.Status)
		}
	})

	t.Run("critical dyno status due to error rate", func(t *testing.T) {
		app := createTestAppForMetrics()
		app.StartMetricsAggregator()

		for i := 0; i < 9; i++ {
			log := createTestParsedLog(500, "GET", "/test", "", "web.1", 100*time.Millisecond, false)
			app.MetricChan <- log
		}

		log := createTestParsedLog(200, "GET", "/test", "", "web.1", 100*time.Millisecond, false)
		app.MetricChan <- log

		time.Sleep(100 * time.Millisecond)

		dyno := app.Metric.DynoPerformance["web.1"]
		if dyno.Status != "critical" {
			t.Errorf("Expected critical status for high error rate, got %s", dyno.Status)
		}
		if dyno.ErrorRate != 90.0 {
			t.Errorf("Expected 90%% error rate, got %f", dyno.ErrorRate)
		}
	})
}

func TestMetricsAggregator_Integration(t *testing.T) {
	t.Run("full metrics flow", func(t *testing.T) {
		app := createTestAppForMetrics()
		app.StartMetricsAggregator()

		logs := []*ParsedLog{
			createTestParsedLog(200, "GET", "/api/users", "192.168.1.1", "web.1", 100*time.Millisecond, false),
			createTestParsedLog(201, "POST", "/api/users", "10.0.0.1", "web.1", 200*time.Millisecond, false),
			createTestParsedLog(404, "GET", "/api/missing", "192.168.1.2", "web.2", 50*time.Millisecond, false),
			createTestParsedLog(500, "POST", "/api/error", "192.168.1.1", "web.1", 2*time.Second, true),
		}

		for _, log := range logs {
			app.MetricChan <- log
		}

		time.Sleep(200 * time.Millisecond)

		if app.Metric.TotalRequests != 4 {
			t.Errorf("Expected 4 total requests, got %d", app.Metric.TotalRequests)
		}

		if app.Metric.Status2xx != 2 {
			t.Errorf("Expected 2 2xx responses, got %d", app.Metric.Status2xx)
		}

		if app.Metric.Status4xx != 1 {
			t.Errorf("Expected 1 4xx response, got %d", app.Metric.Status4xx)
		}

		if app.Metric.Status5xx != 1 {
			t.Errorf("Expected 1 5xx response, got %d", app.Metric.Status5xx)
		}

		if app.Metric.SlowRequestCount != 1 {
			t.Errorf("Expected 1 slow request, got %d", app.Metric.SlowRequestCount)
		}

		if app.Metric.GetRequests != 2 {
			t.Errorf("Expected 2 GET requests, got %d", app.Metric.GetRequests)
		}

		if app.Metric.PostRequests != 2 {
			t.Errorf("Expected 2 POST requests, got %d", app.Metric.PostRequests)
		}

		if app.Metric.SuccessRate != 50.0 {
			t.Errorf("Expected 50%% success rate, got %f", app.Metric.SuccessRate)
		}

		if app.Metric.ErrorRate != 50.0 {
			t.Errorf("Expected 50%% error rate, got %f", app.Metric.ErrorRate)
		}

		if app.Metric.TopEndpoints["/api/users"] != 2 {
			t.Errorf("Expected 2 requests to /api/users, got %d", app.Metric.TopEndpoints["/api/users"])
		}

		web1Dyno := app.Metric.DynoPerformance["web.1"]
		if web1Dyno.RequestCount != 3 {
			t.Errorf("Expected web.1 to have 3 requests, got %d", web1Dyno.RequestCount)
		}

		web2Dyno := app.Metric.DynoPerformance["web.2"]
		if web2Dyno.RequestCount != 1 {
			t.Errorf("Expected web.2 to have 1 request, got %d", web2Dyno.RequestCount)
		}

		if app.Metric.RequestsPerSecond <= 0 {
			t.Errorf("Expected positive requests per second, got %f", app.Metric.RequestsPerSecond)
		}
	})
}

func BenchmarkApp_StartMetricsAggregator(b *testing.B) {
	app := createTestAppForMetrics()
	app.StartMetricsAggregator()

	log := createTestParsedLog(200, "GET", "/api/test", "192.168.1.1", "web.1", 100*time.Millisecond, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.MetricChan <- log
	}

	time.Sleep(100 * time.Millisecond)
}

func BenchmarkApp_calculatePercentiles(b *testing.B) {
	app := createTestAppForMetrics()
	app.Metric = &Metric{}

	responseTimes := make([]time.Duration, 1000)
	for i := 0; i < 1000; i++ {
		responseTimes[i] = time.Duration(i+1) * time.Millisecond
	}

	aggregator := &MetricsAggregator{
		responseTimes: responseTimes,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.calculatePercentiles(aggregator)
	}
}

func BenchmarkApp_generateAlerts(b *testing.B) {
	app := createTestAppForMetrics()
	app.Metric = &Metric{
		ErrorRate:       15.0,
		P95ResponseTime: 6 * time.Second,
		ActiveAlerts:    []Alert{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.generateAlerts()
	}
}

func BenchmarkApp_updateChannelHealth(b *testing.B) {
	app := createTestAppForMetrics()
	app.Metric = &Metric{}
	app.RawLogChan = make(chan []byte, 100)
	app.ParsedLogChan = make(chan *ParsedLog, 50)

	for i := 0; i < 25; i++ {
		app.RawLogChan <- []byte("test")
	}

	for i := 0; i < 10; i++ {
		app.ParsedLogChan <- &ParsedLog{}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.updateChannelHealth()
	}
}
