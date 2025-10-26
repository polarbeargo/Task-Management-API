package cache

import (
	"errors"
	"sync"
	"time"
)

type CircuitBreakerState int

const (
	CircuitBreakerClosed CircuitBreakerState = iota
	CircuitBreakerOpen
	CircuitBreakerHalfOpen
)

type CircuitBreaker struct {
	mu              sync.RWMutex
	state           CircuitBreakerState
	failureCount    int
	successCount    int
	lastFailureTime time.Time

	maxFailures      int
	timeout          time.Duration
	halfOpenMaxCalls int
}

type CircuitBreakerConfig struct {
	MaxFailures      int           `json:"max_failures"`
	Timeout          time.Duration `json:"timeout"`
	HalfOpenMaxCalls int           `json:"half_open_max_calls"`
}

func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		MaxFailures:      5,
		Timeout:          30 * time.Second,
		HalfOpenMaxCalls: 3,
	}
}

func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	if config == nil {
		config = DefaultCircuitBreakerConfig()
	}

	return &CircuitBreaker{
		state:            CircuitBreakerClosed,
		maxFailures:      config.MaxFailures,
		timeout:          config.Timeout,
		halfOpenMaxCalls: config.HalfOpenMaxCalls,
	}
}

var (
	ErrCircuitBreakerOpen = errors.New("circuit breaker is open")
	ErrTooManyRequests    = errors.New("too many requests")
)

func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.allow() {
		return ErrCircuitBreakerOpen
	}

	err := fn()

	if err != nil {
		cb.recordFailure()
		return err
	}

	cb.recordSuccess()
	return nil
}

func (cb *CircuitBreaker) allow() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case CircuitBreakerClosed:
		return true
	case CircuitBreakerOpen:
		return cb.shouldAttemptReset()
	case CircuitBreakerHalfOpen:
		return cb.successCount < cb.halfOpenMaxCalls
	default:
		return false
	}
}

func (cb *CircuitBreaker) shouldAttemptReset() bool {
	return time.Since(cb.lastFailureTime) >= cb.timeout
}

func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case CircuitBreakerClosed:
		if cb.failureCount >= cb.maxFailures {
			cb.state = CircuitBreakerOpen
		}
	case CircuitBreakerHalfOpen:
		cb.state = CircuitBreakerOpen
		cb.successCount = 0
	}
}

func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitBreakerClosed:
		cb.failureCount = 0
	case CircuitBreakerHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.halfOpenMaxCalls {
			cb.state = CircuitBreakerClosed
			cb.failureCount = 0
			cb.successCount = 0
		}
	case CircuitBreakerOpen:
		if cb.shouldAttemptReset() {
			cb.state = CircuitBreakerHalfOpen
			cb.successCount = 1
		}
	}
}

func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	stateName := "closed"
	switch cb.state {
	case CircuitBreakerOpen:
		stateName = "open"
	case CircuitBreakerHalfOpen:
		stateName = "half-open"
	}

	return map[string]interface{}{
		"state":           stateName,
		"failure_count":   cb.failureCount,
		"success_count":   cb.successCount,
		"last_failure":    cb.lastFailureTime.Unix(),
		"max_failures":    cb.maxFailures,
		"timeout_seconds": cb.timeout.Seconds(),
	}
}
