package internal

import (
	"sync"
	"time"
)

//Security ToDos
/*
anti DOS of course

Auth from heroku, idk - research if any form of keys/tokens provided

Rate limits - Buckets (Research)- avoid making it a bottle neck

Context cancellation to drain channels better

*unrelated - logging


*/
type TokenBucket struct {
	cap        int64
	tokens     int64
	refillRate int64
	lastRefill time.Time
	mu         sync.Mutex
}

type RateLimiterMap struct {
	buckets    map[string]*TokenBucket
	mu         sync.RWMutex
	cap        int64
	refillRate int64
}

func NewRateLimiterMap(capacity, rR int64) *RateLimiterMap {
	return &RateLimiterMap{
		buckets:    make(map[string]*TokenBucket),
		cap:        capacity,
		refillRate: rR,
	}
}

func (rlm *RateLimiterMap) GetBucket(apiKey string) *TokenBucket {
	rlm.mu.RLock()
	bucket, exists := rlm.buckets[apiKey]
	rlm.mu.RUnlock()

	if !exists {
		rlm.mu.Lock()
		if bucket, exists = rlm.buckets[apiKey]; !exists {
			bucket = &TokenBucket{
				cap:        rlm.cap,
				tokens:     rlm.cap,
				refillRate: rlm.refillRate,
				lastRefill: time.Now(),
			}
			rlm.buckets[apiKey] = bucket
		}
		rlm.mu.Unlock()
	}
	return bucket
}

func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tokensToAdd := int64(elapsed * float64(tb.refillRate))

	tb.tokens = min(tb.cap, tb.tokens+tokensToAdd)
	tb.lastRefill = now

	if tb.tokens > 0 {
		tb.tokens--
		return true
	}
	return false
}
