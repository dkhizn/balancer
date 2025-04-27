package ratelimit

import (
	"sync"
	"time"

	db "github.com/ternaryinvalid/balancer/internal/database"
)

type TokenBucket struct {
	capacity    int
	rate        int // скорость пополнения токенов
	tokenChan   chan struct{}
	closeChan   chan struct{}
	lastUpdated time.Time
}

type RateLimiter struct {
	buckets  map[string]*TokenBucket
	db       *db.DB
	updateCh chan string
	mu       sync.RWMutex
}

func (tb *TokenBucket) tryAllow() bool {
	select {
	case <-tb.tokenChan:
		return true
	default:
		return false
	}
}

func NewRateLimiter(db *db.DB) *RateLimiter {
	rl := &RateLimiter{
		db:       db,
		buckets:  make(map[string]*TokenBucket),
		updateCh: make(chan string, 100),
	}

	go rl.ruleUpdater()
	go rl.cleanupWorker()

	return rl
}

func (rl *RateLimiter) Allow(clientID string) bool {
	rl.mu.RLock()
	bucket, exists := rl.buckets[clientID]
	rl.mu.RUnlock()

	if !exists {
		return rl.initBucket(clientID)
	}

	select {
	case <-bucket.tokenChan:
		return true
	default:
		return false
	}
}

func (rl *RateLimiter) initBucket(clientID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if bucket, exists := rl.buckets[clientID]; exists {
		return bucket.tryAllow()
	}

	capacity, rate, err := rl.db.GetRateLimitRule(clientID)
	if err != nil {
		return true
	}

	bucket := &TokenBucket{
		capacity:    capacity,
		rate:        rate,
		tokenChan:   make(chan struct{}, capacity),
		closeChan:   make(chan struct{}),
		lastUpdated: time.Now(),
	}

	for i := 0; i < capacity; i++ {
		bucket.tokenChan <- struct{}{}
	}

	go rl.refillWorker(clientID, bucket)

	rl.buckets[clientID] = bucket
	return bucket.tryAllow()
}

func (rl *RateLimiter) refillWorker(clientID string, b *TokenBucket) {
	ticker := time.NewTicker(time.Second / time.Duration(b.rate))
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			select {
			case b.tokenChan <- struct{}{}:
				b.lastUpdated = time.Now()
			default:

			}
		case <-b.closeChan:
			return
		}
	}
}

func (rl *RateLimiter) ruleUpdater() {
	for clientID := range rl.updateCh {
		rl.mu.Lock()
		bucket, exists := rl.buckets[clientID]
		if !exists {
			rl.mu.Unlock()
			continue
		}

		capacity, rate, err := rl.db.GetRateLimitRule(clientID)
		if err != nil {
			rl.mu.Unlock()
			continue
		}

		close(bucket.closeChan)
		newBucket := &TokenBucket{
			capacity:    capacity,
			rate:        rate,
			tokenChan:   make(chan struct{}, capacity),
			closeChan:   make(chan struct{}),
			lastUpdated: time.Now(),
		}

		remaining := len(bucket.tokenChan)
		for i := 0; i < remaining && i < capacity; i++ {
			newBucket.tokenChan <- struct{}{}
		}

		rl.buckets[clientID] = newBucket
		go rl.refillWorker(clientID, newBucket)
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) cleanupWorker() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		for clientID, bucket := range rl.buckets {
			if time.Since(bucket.lastUpdated) > 24*time.Hour {
				close(bucket.closeChan)
				delete(rl.buckets, clientID)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) SetRule(clientID string, capacity, rate int) error {
	err := rl.db.SetRateLimitRule(clientID, capacity, rate)
	if err != nil {
		return err
	}

	rl.updateCh <- clientID
	return nil
}
