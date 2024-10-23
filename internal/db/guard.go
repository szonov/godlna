package db

import (
	"math/rand"
	"sync"
	"time"
)

// ref: https://github.com/EagleChen/mapmutex/blob/master/mutex.go

// Guard protect scanner from multiply scan in the same time for one directory
type Guard struct {
	locks     map[string]bool
	m         *sync.Mutex
	maxRetry  int
	maxDelay  float64 // in nanosecond
	baseDelay float64 // in nanosecond
	factor    float64
	jitter    float64
}

func (m *Guard) TryLock(key string) (gotLock bool) {
	for i := 0; i < m.maxRetry; i++ {
		m.m.Lock()
		if _, ok := m.locks[key]; ok { // if locked
			m.m.Unlock()
			time.Sleep(m.backoff(i))
		} else { // if unlocked lock it
			m.locks[key] = true
			m.m.Unlock()
			return true
		}
	}
	return false
}

func (m *Guard) Unlock(key string) {
	m.m.Lock()
	delete(m.locks, key)
	m.m.Unlock()
}

func (m *Guard) backoff(retries int) time.Duration {
	if retries == 0 {
		return time.Duration(m.baseDelay) * time.Nanosecond
	}
	backoff, max_ := m.baseDelay, m.maxDelay
	for backoff < max_ && retries > 0 {
		backoff *= m.factor
		retries--
	}
	if backoff > max_ {
		backoff = max_
	}
	backoff *= 1 + m.jitter*(rand.Float64()*2-1)
	if backoff < 0 {
		return 0
	}
	return time.Duration(backoff) * time.Nanosecond
}

func NewGuard() *Guard {
	return &Guard{
		locks:     make(map[string]bool),
		m:         &sync.Mutex{},
		maxRetry:  200,
		maxDelay:  100000000, // 0.1 second
		baseDelay: 10,        // 10 nanosecond
		factor:    1.1,
		jitter:    0.2,
	}
}
