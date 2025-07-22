package internal

import (
	"log"
	"strconv"
	"time"

	ip2 "github.com/ip2location/ip2location-go"
)

const (
	HealthyThreshold = 250 * time.Millisecond
	MediumThreshold  = 600 * time.Millisecond
)

// Ring buffer to make use of frame if to avoid dupes incase of retries from logplex
func NewDedupeCache(s int) *DedupeCache {
	return &DedupeCache{
		Buffer:   make([]string, s),
		Lookup:   make(map[string]struct{}),
		Size:     s,
		WritePos: 0,
	}
}
func (d *DedupeCache) Add(msgId string) bool {
	if _, exists := d.Lookup[msgId]; exists {
		return false
	}

	//evict before insert
	e := d.Buffer[d.WritePos]
	if e != "" {
		delete(d.Lookup, e)
	}
	d.Buffer[d.WritePos] = msgId
	d.Lookup[msgId] = struct{}{}
	d.WritePos = (d.WritePos + 1) % d.Size

	return true

}
func ParseDuration(s string) (t time.Duration) {
	t, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return t

}

func BuildParsedLog(d map[string]string) *ParsedLog {
	size, err := strconv.Atoi(d["bytes"])
	if err != nil {
		size = 0 // Default value instead of ignoring error
	}

	status, err := strconv.Atoi(d["status"])
	if err != nil {
		status = 0 // Default value for invalid status
	}

	responseTime, err := time.ParseDuration(d["service"])
	if err != nil {
		responseTime = 0
		if d["service"] != "" {
			log.Printf("Invalid service duration: %s", d["service"])
		}
	}
	connectTime, err := time.ParseDuration(d["connect"])
	if err != nil {
		connectTime = 0
		if d["connect"] != "" {
			log.Printf("Invalid connect duration: %s", d["connect"])
		}
	}
	var success bool
	var isSlow bool
	if status < 400 {
		success = true
	} else {
		success = false
	}
	threshold := ClassifyResTime(responseTime)
	if threshold == "medium" {
		isSlow = true
	} else {
		isSlow = false
	}
	timestamp, err := time.Parse(time.RFC3339Nano, d["timestamp"])
	if err != nil {
		if d["timestamp"] != "" {
			log.Printf("Invalid timestamp format: %s, error: %v", d["timestamp"], err)
		}
		// Return zero time instead of nil
		timestamp = time.Time{}
	}

	return &ParsedLog{
		Time:         timestamp,
		Level:        d["at"],
		Size:         size,
		ConnectTime:  connectTime,
		SourceDyno:   d["dyno"],
		SourceIp:     d["fwd"],
		Host:         d["host"],
		Method:       d["method"],
		Path:         d["path"],
		Protocol:     d["protocol"],
		ResponseTime: responseTime,
		Status:       status,
		Success:      success,
		Threshold:    threshold,
		IsSlow:       isSlow,
	}

}

func ClassifyResTime(s time.Duration) string {
	switch {
	case s <= HealthyThreshold:
		return "healthy"

	case s <= MediumThreshold:
		return "medium"
	default:
		return "critical"
	}

}
func (a *App) fingerPrintIp(ip string) ip2.IP2Locationrecord {
	if a.GeoDb == nil {
		return ip2.IP2Locationrecord{}
	}

	result, err := a.GeoDb.Get_all(ip)
	if err != nil {
		return ip2.IP2Locationrecord{}
	}
	return result
}
