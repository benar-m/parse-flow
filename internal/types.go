package internal

import "time"

type App struct {
	Dc            *DedupeCache
	RawLogChan    chan []byte
	ParsedLogChan chan *ParsedLog
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
