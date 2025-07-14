package internal

import (
	"sort"
	"sync"
	"time"
)

//Read from Parsed Log

//Brainstorm
/*
The idea is to have a central metrics channel
a routine comes across a 200, updates there
to avoid races, only one go routine will own that metrics channel, and the rest send data there and continue

&{2025-07-09 13:37:42.123456 +0000 +0000 info 1345 1ms web.1 "197.248.10.42" myapp.herokuapp.com GET "/login" https  23ms 200 true healthy false}map[at:info bytes:1345 connect:1ms dyno:web.1 fwd:"197.248.10.42" host:myapp.herokuapp.com method:GET path:"/login" protocol:https request_id:123abc-456def service:23ms status:200 timestamp:2025-07-09T13:37:42.123456+00:00]
&{2025-07-09 13:37:42.123456 +0000 +0000 info 1345 1ms web.1 "197.248.10.42" myapp.herokuapp.com GET "/login" https  23ms 200 true healthy false}

resort to an actor model
A single go routines owns a metrics struct, and there exists a channel to wwhich
 parsers send data to (for now the function will be in metrics.go)
the go routine then updates its own struct
include a mutex lock since visualizer will need to read that struct to render a dashboard.

*/
//Notes for later ; alert clearing on resolution

// MetricsAggregator holds additional data needed for calculations
type MetricsAggregator struct {
	responseTimes []time.Duration
	dynoErrors    map[string]int64 // Track errors per dyno
	startTime     time.Time
	mu            sync.RWMutex
}

// A function to a go routine that will own a metrics instance
func (a *App) StartMetricsAggregator() {
	a.Metric = &Metric{
		Timestamp:       time.Now(),
		TopCountries:    make(map[string]int64),
		DynoPerformance: make(map[string]DynoMetric),
		TopEndpoints:    make(map[string]int64),
		ActiveAlerts:    []Alert{},
	}

	// Initialize aggregator
	aggregator := &MetricsAggregator{
		responseTimes: make([]time.Duration, 0, 1000), // Preallocate for efficiency
		dynoErrors:    make(map[string]int64),
		startTime:     time.Now(),
	}

	//classify requests and increment their counters
	go func() {
		for l := range a.ParsedLogChan {
			aggregator.mu.Lock()

			//classify status code
			switch {
			case l.Status >= 200 && l.Status < 300:
				a.Metric.Status2xx++
			case l.Status >= 300 && l.Status < 400:
				a.Metric.Status3xx++
			case l.Status >= 400 && l.Status < 500:
				a.Metric.Status4xx++
				// rrack error per dyno for perf
				if l.SourceDyno != "" {
					aggregator.dynoErrors[l.SourceDyno]++
				}
			case l.Status >= 500:
				a.Metric.Status5xx++
				if l.SourceDyno != "" {
					aggregator.dynoErrors[l.SourceDyno]++
				}
			}

			//count all requests as valid
			a.Metric.TotalRequests++

			// ++ slow requests
			if l.IsSlow {
				a.Metric.SlowRequestCount++
			}

			// Classify methods
			switch l.Method {
			case "GET":
				a.Metric.GetRequests++
			case "POST":
				a.Metric.PostRequests++
			case "PUT":
				a.Metric.PutRequests++
			case "DELETE":
				a.Metric.DeleteRequests++
			default:
				a.Metric.OtherRequests++
			}

			// Track top 5 endpoints
			if len(a.Metric.TopEndpoints) < 5 || a.Metric.TopEndpoints[l.Path] > 0 {
				a.Metric.TopEndpoints[l.Path]++
			}

			if l.SourceIp != "" && a.GeoDb != nil {
				ip := l.SourceIp
				if len(ip) > 2 && ip[0] == '"' && ip[len(ip)-1] == '"' {
					ip = ip[1 : len(ip)-1]
				}

				geoRecord := a.fingerPrintIp(ip)
				if geoRecord.Country_short != "" {
					// Limit to top 10 countries to control memory usage
					if len(a.Metric.TopCountries) < 5 || a.Metric.TopCountries[geoRecord.Country_short] > 0 {
						a.Metric.TopCountries[geoRecord.Country_short]++
					}
				}
			}

			if l.SourceDyno != "" {
				dynoMetric, exists := a.Metric.DynoPerformance[l.SourceDyno]
				if !exists {
					dynoMetric = DynoMetric{
						Name:            l.SourceDyno,
						RequestCount:    0,
						AvgResponseTime: 0,
						ErrorRate:       0,
						Status:          "healthy",
					}
				}

				dynoMetric.RequestCount++

				// calc error rate for focus dyno
				errorCount := aggregator.dynoErrors[l.SourceDyno]
				dynoMetric.ErrorRate = (float64(errorCount) / float64(dynoMetric.RequestCount)) * 100

				if dynoMetric.AvgResponseTime == 0 {
					dynoMetric.AvgResponseTime = l.ResponseTime
				} else {
					// Simple moving average
					alpha := 0.1
					dynoMetric.AvgResponseTime = time.Duration(float64(dynoMetric.AvgResponseTime)*(1-alpha) + float64(l.ResponseTime)*alpha)
				}

				// per dyno health
				if dynoMetric.ErrorRate > 10.0 || dynoMetric.AvgResponseTime > 2*time.Second {
					dynoMetric.Status = "critical"
				} else if dynoMetric.ErrorRate > 5.0 || dynoMetric.AvgResponseTime > 1*time.Second {
					dynoMetric.Status = "warning"
				} else {
					dynoMetric.Status = "healthy"
				}

				a.Metric.DynoPerformance[l.SourceDyno] = dynoMetric
			}

			// response times for percentile calculations
			aggregator.responseTimes = append(aggregator.responseTimes, l.ResponseTime)

			// Calculate percentiles every 300 requests
			if len(aggregator.responseTimes) >= 300 {
				a.calculatePercentiles(aggregator)
			}

			// success/error rates
			if a.Metric.TotalRequests > 0 {
				a.Metric.SuccessRate = (float64(a.Metric.Status2xx) / float64(a.Metric.TotalRequests)) * 100
				a.Metric.ErrorRate = (float64(a.Metric.Status4xx+a.Metric.Status5xx) / float64(a.Metric.TotalRequests)) * 100
			}

			// calculate requests per second
			elapsed := time.Since(aggregator.startTime).Seconds()
			if elapsed > 0 {
				a.Metric.RequestsPerSecond = float64(a.Metric.TotalRequests) / elapsed
			}

			if len(aggregator.responseTimes) > 0 {
				var total time.Duration
				for _, rt := range aggregator.responseTimes {
					total += rt
				}
				a.Metric.AvgResponseTime = total / time.Duration(len(aggregator.responseTimes))
			}

			a.generateAlerts()
			a.updateChannelHealth()
			a.Metric.Timestamp = time.Now()
			aggregator.mu.Unlock()
		}
	}()
}

func (a *App) calculatePercentiles(aggregator *MetricsAggregator) {
	if len(aggregator.responseTimes) == 0 {
		return
	}
	sortedTimes := make([]time.Duration, len(aggregator.responseTimes))
	copy(sortedTimes, aggregator.responseTimes)
	sort.Slice(sortedTimes, func(i, j int) bool {
		return sortedTimes[i] < sortedTimes[j]
	})
	length := len(sortedTimes)
	a.Metric.P50ResponseTime = sortedTimes[length*50/100]
	a.Metric.P95ResponseTime = sortedTimes[length*95/100]
	a.Metric.P99ResponseTime = sortedTimes[length*99/100]
	if len(aggregator.responseTimes) > 1000 {
		aggregator.responseTimes = aggregator.responseTimes[len(aggregator.responseTimes)-500:]
	}
}

func (a *App) generateAlerts() {
	currentTime := time.Now()

	//delete alerts
	var activeAlerts []Alert
	for _, alert := range a.Metric.ActiveAlerts {
		if !alert.Resolved || time.Since(alert.Timestamp) < time.Hour {
			activeAlerts = append(activeAlerts, alert)
		}
	}
	a.Metric.ActiveAlerts = activeAlerts

	// high error rate alert
	if a.Metric.ErrorRate > 5.0 {
		alert := Alert{
			Type:      "high_error_rate",
			Severity:  "warning",
			Message:   "Error rate is above 5%",
			Timestamp: currentTime,
			Resolved:  false,
		}
		if a.Metric.ErrorRate > 10.0 {
			alert.Severity = "critical"
			alert.Message = "Error rate is above 10%"
		}

		//avoid same alerts
		found := false
		for i, existingAlert := range a.Metric.ActiveAlerts {
			if existingAlert.Type == "high_error_rate" && !existingAlert.Resolved {
				a.Metric.ActiveAlerts[i] = alert // Update existing
				found = true
				break
			}
		}
		if !found {
			a.Metric.ActiveAlerts = append(a.Metric.ActiveAlerts, alert)
		}
	}

	// slow response alert
	if a.Metric.P95ResponseTime > 2*time.Second {
		alert := Alert{
			Type:      "slow_response",
			Severity:  "warning",
			Message:   "P95 response time is above 2 seconds",
			Timestamp: currentTime,
			Resolved:  false,
		}
		if a.Metric.P95ResponseTime > 5*time.Second {
			alert.Severity = "critical"
			alert.Message = "P95 response time is above 5 seconds"
		}
		found := false
		for i, existingAlert := range a.Metric.ActiveAlerts {
			if existingAlert.Type == "slow_response" && !existingAlert.Resolved {
				a.Metric.ActiveAlerts[i] = alert
				found = true
				break
			}
		}
		if !found {
			a.Metric.ActiveAlerts = append(a.Metric.ActiveAlerts, alert)
		}
	}
}

func (a *App) updateChannelHealth() {
	rawChanCap := cap(a.RawLogChan)
	rawChanLen := len(a.RawLogChan)
	parsedChanCap := cap(a.ParsedLogChan)
	parsedChanLen := len(a.ParsedLogChan)

	a.Metric.ChannelHealth = ChannelHealth{
		RawLogChanUsage:    (float64(rawChanLen) / float64(rawChanCap)) * 100,
		ParsedLogChanUsage: (float64(parsedChanLen) / float64(parsedChanCap)) * 100,
		ProcessingRate:     a.Metric.RequestsPerSecond,
		BacklogSize:        rawChanLen + parsedChanLen,
	}
}
