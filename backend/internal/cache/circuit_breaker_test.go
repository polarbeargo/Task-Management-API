package cache

import (
	"fmt"
	"testing"
	"time"
)

func TestCircuitBreakerBasicFlow(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:      3,
		Timeout:          100 * time.Millisecond,
		HalfOpenMaxCalls: 2,
	}

	cb := NewCircuitBreaker(config)

	
	if cb.GetState() != CircuitBreakerClosed {
		t.Errorf("Expected initial state to be Closed, got %v", cb.GetState())
	}

	
	err := cb.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if cb.GetState() != CircuitBreakerClosed {
		t.Errorf("Expected state to remain Closed after success, got %v", cb.GetState())
	}
}

func TestCircuitBreakerFailureTransition(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:      2,
		Timeout:          100 * time.Millisecond,
		HalfOpenMaxCalls: 2,
	}

	cb := NewCircuitBreaker(config)

	
	err := cb.Execute(func() error {
		return fmt.Errorf("operation failed")
	})
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if cb.GetState() != CircuitBreakerClosed {
		t.Errorf("Expected state to be Closed after first failure, got %v", cb.GetState())
	}

	
	err = cb.Execute(func() error {
		return fmt.Errorf("operation failed again")
	})
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if cb.GetState() != CircuitBreakerOpen {
		t.Errorf("Expected state to be Open after reaching failure threshold, got %v", cb.GetState())
	}
}

func TestCircuitBreakerOpenState(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:      1,
		Timeout:          100 * time.Millisecond,
		HalfOpenMaxCalls: 2,
	}

	cb := NewCircuitBreaker(config)

	
	cb.Execute(func() error {
		return fmt.Errorf("failure")
	})

	if cb.GetState() != CircuitBreakerOpen {
		t.Errorf("Expected state to be Open, got %v", cb.GetState())
	}

	
	err := cb.Execute(func() error {
		t.Error("Operation should not be executed when circuit is open")
		return nil
	})

	if err != ErrCircuitBreakerOpen {
		t.Errorf("Expected ErrCircuitBreakerOpen, got %v", err)
	}
}

func TestCircuitBreakerHalfOpenTransition(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:      1,
		Timeout:          50 * time.Millisecond,
		HalfOpenMaxCalls: 2,
	}

	cb := NewCircuitBreaker(config)

	
	cb.Execute(func() error {
		return fmt.Errorf("failure")
	})

	if cb.GetState() != CircuitBreakerOpen {
		t.Errorf("Expected state to be Open, got %v", cb.GetState())
	}

	
	time.Sleep(60 * time.Millisecond)

	
	executed := false
	err := cb.Execute(func() error {
		executed = true
		return nil 
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !executed {
		t.Error("Expected operation to be executed in half-open state")
	}
	
	state := cb.GetState()
	if state != CircuitBreakerClosed && state != CircuitBreakerHalfOpen {
		t.Errorf("Expected state to be Closed or HalfOpen after successful half-open execution, got %v", state)
	}
}

func TestCircuitBreakerTimeout(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:      1,
		Timeout:          50 * time.Millisecond,
		HalfOpenMaxCalls: 2,
	}

	cb := NewCircuitBreaker(config)

	
	cb.Execute(func() error {
		return fmt.Errorf("failure")
	})

	if cb.GetState() != CircuitBreakerOpen {
		t.Errorf("Expected state to be Open, got %v", cb.GetState())
	}

	
	time.Sleep(60 * time.Millisecond)

	
	executed := false
	err := cb.Execute(func() error {
		executed = true
		return nil 
	})

	if err != nil {
		t.Errorf("Expected no error after timeout, got %v", err)
	}
	if !executed {
		t.Error("Expected operation to be executed after timeout")
	}
}

func TestCircuitBreakerConcurrency(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:      5,
		Timeout:          100 * time.Millisecond,
		HalfOpenMaxCalls: 3,
	}

	cb := NewCircuitBreaker(config)

	
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				cb.Execute(func() error {
					
					if (id+j)%3 == 0 {
						return fmt.Errorf("failure %d-%d", id, j)
					}
					return nil
				})
			}
			done <- true
		}(i)
	}

	
	for i := 0; i < 10; i++ {
		<-done
	}

	
	err := cb.Execute(func() error {
		return nil
	})

	
	if err != nil && err != ErrCircuitBreakerOpen {
		t.Errorf("Unexpected error after concurrent operations: %v", err)
	}
}
