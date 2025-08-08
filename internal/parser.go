package internal

import (
	"log"
	"sort"
	"strings"
	"time"
)

func (a *App) ParseLog(logByte []byte) map[string]string {
	logString := string(logByte)
	fields := strings.Fields(logString)
	logParts := make(map[string]string)

	for _, f := range fields {
		if strings.Contains(f, "=") {
			parts := strings.SplitN(f, "=", 2)
			k := parts[0]
			v := parts[1]
			logParts[k] = v

		}
	}
	f := strings.SplitN(logString, " ", 2)
	if len(f) < 2 {
		log.Println("Malformed Request Received")
		return map[string]string{}
	}
	timeStampstr := f[0]
	logParts["timestamp"] = timeStampstr
	parsedlog := BuildParsedLog(logParts)

	// Always send parsed logs (BuildParsedLog now returns valid struct with defaults)
	a.ParsedLogChan <- parsedlog

	// Track for percentile calculation
	a.trackForPercentiles(parsedlog)

	return logParts

}

func (a *App) trackForPercentiles(parsedLog *ParsedLog) {
	if parsedLog.ResponseTime == 0 {
		return // Skip if no response time
	}

	a.PercentileTracker.Mutex.Lock()
	defer a.PercentileTracker.Mutex.Unlock()

	a.PercentileTracker.ResponseTimes = append(a.PercentileTracker.ResponseTimes, parsedLog.ResponseTime)
	a.PercentileTracker.RequestCount++

	now := time.Now()
	timeSinceLastCalc := now.Sub(a.PercentileTracker.LastCalcTime)
	if timeSinceLastCalc >= 10*time.Second || a.PercentileTracker.RequestCount >= 300 {
		a.calculateAndUpdatePercentiles()

		// Reset tracking
		a.PercentileTracker.ResponseTimes = []time.Duration{}
		a.PercentileTracker.RequestCount = 0
		a.PercentileTracker.LastCalcTime = now
	}
}

func (a *App) calculateAndUpdatePercentiles() {
	if len(a.PercentileTracker.ResponseTimes) == 0 {
		return
	}
	times := make([]time.Duration, len(a.PercentileTracker.ResponseTimes))
	copy(times, a.PercentileTracker.ResponseTimes)
	sort.Slice(times, func(i, j int) bool {
		return times[i] < times[j]
	})

	// Calculate percentiles
	p50 := calculatePercentile(times, 0.50)
	p95 := calculatePercentile(times, 0.95)
	p99 := calculatePercentile(times, 0.99)

	// Update metrics
	a.MetricsMu.Lock()
	a.Metric.P50ResponseTime = p50
	a.Metric.P95ResponseTime = p95
	a.Metric.P99ResponseTime = p99
	a.MetricsMu.Unlock()

	log.Printf("Percentiles updated: P50=%v, P95=%v, P99=%v (from %d requests)",
		p50, p95, p99, len(times))
}

func calculatePercentile(sortedTimes []time.Duration, percentile float64) time.Duration {
	if len(sortedTimes) == 0 {
		return 0
	}

	index := int(percentile * float64(len(sortedTimes)-1))
	if index >= len(sortedTimes) {
		index = len(sortedTimes) - 1
	}

	return sortedTimes[index]
}

// Reads from RawLogChan and Parses
func (a *App) ParserWorker() {
	for logBytes := range a.RawLogChan {
		a.ParseLog(logBytes)
	}
}
