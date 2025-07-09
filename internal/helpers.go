package internal

import (
	"fmt"
	"log"
	"strconv"
	"time"
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
	size, _ := strconv.Atoi(d["bytes"])
	status, _ := strconv.Atoi(d["status"])
	responseTime, err := time.ParseDuration(d["service"])
	if err != nil {
		responseTime = 0
		log.Println("bad duration parsed")
	}
	connectTime, err := time.ParseDuration(d["connect"])
	if err != nil {
		log.Println("bad duration parsed")
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
	time, err := time.Parse(time.RFC3339Nano, d["timestamp"])
	if err != nil {
		fmt.Printf("time %v error %v", time, err)
		return &ParsedLog{}
	}

	return &ParsedLog{
		Time:         time,
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
