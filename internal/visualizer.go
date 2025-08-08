package internal

import (
	"crypto/hmac"
	"encoding/json"
	"maps"
	"net/http"
	"os"
	"strconv"
)

func (a *App) GetMetricsSnapshot() *Metric {
	a.MetricsMu.RLock()
	defer a.MetricsMu.RUnlock()

	if a.Metric == nil {
		return &Metric{}
	}

	// deep copy of the metrics
	snapshot := &Metric{
		Timestamp:         a.Metric.Timestamp,
		TotalRequests:     a.Metric.TotalRequests,
		RequestsPerSecond: a.Metric.RequestsPerSecond,
		SuccessRate:       a.Metric.SuccessRate,
		ErrorRate:         a.Metric.ErrorRate,
		AvgResponseTime:   a.Metric.AvgResponseTime,
		P50ResponseTime:   a.Metric.P50ResponseTime,
		P95ResponseTime:   a.Metric.P95ResponseTime,
		P99ResponseTime:   a.Metric.P99ResponseTime,
		SlowRequestCount:  a.Metric.SlowRequestCount,
		Status2xx:         a.Metric.Status2xx,
		Status3xx:         a.Metric.Status3xx,
		Status4xx:         a.Metric.Status4xx,
		Status5xx:         a.Metric.Status5xx,
		GetRequests:       a.Metric.GetRequests,
		PostRequests:      a.Metric.PostRequests,
		PutRequests:       a.Metric.PutRequests,
		DeleteRequests:    a.Metric.DeleteRequests,
		OtherRequests:     a.Metric.OtherRequests,
		ChannelHealth:     a.Metric.ChannelHealth,
	}

	// copy maps and slices
	snapshot.TopCountries = make(map[string]int64)
	maps.Copy(snapshot.TopCountries, a.Metric.TopCountries)

	snapshot.DynoPerformance = make(map[string]DynoMetric)
	maps.Copy(snapshot.DynoPerformance, a.Metric.DynoPerformance)

	snapshot.TopEndpoints = make(map[string]int64)
	maps.Copy(snapshot.TopEndpoints, a.Metric.TopEndpoints)

	snapshot.ActiveAlerts = make([]Alert, len(a.Metric.ActiveAlerts))
	copy(snapshot.ActiveAlerts, a.Metric.ActiveAlerts)

	return snapshot
}

func (a *App) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("X-API-KEY")
	expectedKey := os.Getenv("METRICS_API_KEY")
	if !hmac.Equal([]byte(apiKey), []byte(expectedKey)) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "unauthorized",
		})
		return
	}
	bucket := a.RateLimiter.GetBucket(apiKey)
	if !bucket.Allow() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{
			"error":       "rate limit exceeded",
			"retry_after": "1",
		})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	snapshot := a.GetMetricsSnapshot()

	if err := json.NewEncoder(w).Encode(snapshot); err != nil {
		http.Error(w, "Failed to encode metrics", http.StatusInternalServerError)
		return
	}
}

// HistoricalHandler serves historical metrics data based on percentage-based time ranges
func (a *App) HistoricalHandler(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("X-API-KEY")
	expectedKey := os.Getenv("METRICS_API_KEY")
	if !hmac.Equal([]byte(apiKey), []byte(expectedKey)) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "unauthorized",
		})
		return
	}
	bucket := a.RateLimiter.GetBucket(apiKey)
	if !bucket.Allow() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{
			"error":       "rate limit exceeded",
			"retry_after": "1",
		})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Parse percentage query parameter
	percentageStr := r.URL.Query().Get("percentage")
	if percentageStr == "" {
		http.Error(w, "Missing 'percentage' query parameter", http.StatusBadRequest)
		return
	}

	percentage, err := strconv.ParseFloat(percentageStr, 64)
	if err != nil {
		http.Error(w, "Invalid percentage value", http.StatusBadRequest)
		return
	}

	// Validate (0.138% minimum for 1 hour, 100% maximum for 1 month)
	if percentage < 0.138 || percentage > 100 {
		http.Error(w, "Invalid range", http.StatusBadRequest)
		return
	}

	snapshots, err := a.GetHistoricalSnapshots(percentage)
	if err != nil {
		http.Error(w, "Failed to retrieve historical data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"percentage":     percentage,
		"snapshot_count": len(snapshots),
		"snapshots":      snapshots,
	}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
