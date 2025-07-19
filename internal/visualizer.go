package internal

import (
	"encoding/json"
	"maps"
	"net/http"
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
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	snapshot := a.GetMetricsSnapshot()

	if err := json.NewEncoder(w).Encode(snapshot); err != nil {
		http.Error(w, "Failed to encode metrics", http.StatusInternalServerError)
		return
	}
}
