package internal

import (
	"sync"
	"time"

	ip2 "github.com/ip2location/ip2location-go"
)

type App struct {
	Dc             *DedupeCache
	RawLogChan     chan []byte
	ParsedLogChan  chan *ParsedLog
	GeoDb          *ip2.DB
	Metric         *Metric
	MetricsMu      sync.RWMutex // protect Metric field
	DbWriteChan    chan *Metric
	DbRawWriteChan chan *ParsedLog
	MetricChan     chan *ParsedLog
	RateLimiter    *RateLimiterMap
	Config         *Config
}
type DedupeCache struct {
	Buffer   []string //ring buffer
	Lookup   map[string]struct{}
	Size     int
	WritePos int //cuurent write position
}

type ParsedLog struct {
	Time         time.Time
	Level        string
	Size         int
	ConnectTime  time.Duration
	SourceDyno   string
	SourceIp     string
	Host         string
	Method       string
	Path         string
	Protocol     string
	ReqId        string
	ResponseTime time.Duration
	Status       int
	Success      bool
	Threshold    string
	IsSlow       bool
}
type Metric struct {
	Timestamp         time.Time     `json:"timestamp"`
	TotalRequests     int64         `json:"total_requests"`
	RequestsPerSecond float64       `json:"requests_per_second"`
	SuccessRate       float64       `json:"success_rate"`
	ErrorRate         float64       `json:"error_rate"`
	AvgResponseTime   time.Duration `json:"avg_response_time"`
	P50ResponseTime   time.Duration `json:"p50_response_time"`
	P95ResponseTime   time.Duration `json:"p95_response_time"`
	P99ResponseTime   time.Duration `json:"p99_response_time"`
	SlowRequestCount  int64         `json:"slow_request_count"`
	Status2xx         int64         `json:"status_2xx"`
	Status3xx         int64         `json:"status_3xx"`
	Status4xx         int64         `json:"status_4xx"`
	Status5xx         int64         `json:"status_5xx"`
	GetRequests       int64         `json:"get_requests"`
	PostRequests      int64         `json:"post_requests"`
	PutRequests       int64         `json:"put_requests"`
	DeleteRequests    int64         `json:"delete_requests"`
	OtherRequests     int64         `json:"other_requests"`

	TopCountries map[string]int64 `json:"top_countries"` // Country| count

	DynoPerformance map[string]DynoMetric `json:"dyno_performance"`
	TopEndpoints    map[string]int64      `json:"top_endpoints"`
	ChannelHealth   ChannelHealth         `json:"channel_health"`
	ActiveAlerts    []Alert               `json:"active_alerts"`
}

type DynoMetric struct {
	Name            string        `json:"name"`
	RequestCount    int64         `json:"request_count"`
	AvgResponseTime time.Duration `json:"avg_response_time"`
	ErrorRate       float64       `json:"error_rate"`
	Status          string        `json:"status"` // "healthy", "warning", "critical"
}

type ChannelHealth struct {
	RawLogChanUsage    float64 `json:"raw_log_chan_usage"`    // 0-100%
	ParsedLogChanUsage float64 `json:"parsed_log_chan_usage"` // 0-100%
	ProcessingRate     float64 `json:"processing_rate"`       // logs/sec
	BacklogSize        int     `json:"backlog_size"`
}

type Alert struct {
	Type      string    `json:"type"`     // "higherrorrate", "slowResponse", "dynoDown"
	Severity  string    `json:"severity"` // "warning", "critical"
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Resolved  bool      `json:"resolved"`
}
