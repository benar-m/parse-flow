package internal

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port              string
	AuthToken         string
	DatabasePath      string
	RawLogChanSize    int
	ParsedLogChanSize int
	MetricChanSize    int
	BatchSize         int
	FlushInterval     time.Duration
	SnapshotInterval  time.Duration
}

func LoadConfig() *Config {
	return &Config{
		Port:              getEnv("PORT", "5000"),
		AuthToken:         getEnv("AUTH_TOKEN", ""),
		DatabasePath:      getEnv("DATABASE_PATH", "./logs.db"),
		RawLogChanSize:    getEnvInt("RAW_LOG_CHAN_SIZE", 1000),
		ParsedLogChanSize: getEnvInt("PARSED_LOG_CHAN_SIZE", 1000),
		MetricChanSize:    getEnvInt("METRIC_CHAN_SIZE", 100),
		BatchSize:         getEnvInt("BATCH_SIZE", 100),
		FlushInterval:     getEnvDuration("FLUSH_INTERVAL", 5*time.Second),
		SnapshotInterval:  getEnvDuration("SNAPSHOT_INTERVAL", 1*time.Minute),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}
