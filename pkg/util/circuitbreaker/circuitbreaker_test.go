package circuitbreaker

import (
	"errors"
	"os"
	"testing"
	"time"
)

var (
	now time.Time
	err = errors.New("some error")
)

func setup() {
	now = time.Now()
	nowFunc = func() time.Time {
		return now
	}
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	os.Exit(code)
}

func runSharedCases(t *testing.T, cb *CircuitBreaker) {
	// insert 10 success results
	for i := 0; i < 10; i++ {
		if !cb.AcquirePermission() {
			t.Errorf("acquire permission should succeeded, i = %d", i)
		}
		cb.RecordResult(nil, time.Millisecond)
	}
	// insert 10 failure results
	for i := 0; i < 10; i++ {
		if !cb.AcquirePermission() {
			t.Errorf("acquire permission should succeeded, i = %d", i)
		}
		cb.RecordResult(err, time.Millisecond)
	}

	// state should transit to open now
	if cb.State() != Open {
		t.Errorf("circuit breaker state should be Open")
	}
	if cb.AcquirePermission() {
		t.Errorf("acquire permission should fail")
	}

	// wait 5 seconds for state to transit to half open
	now = now.Add(5 * time.Second)

	// insert 5 success results
	for i := 0; i < 5; i++ {
		if !cb.AcquirePermission() {
			t.Errorf("acquire permission should succeeded, i = %d", i)
		}
		if cb.State() != HalfOpen {
			t.Errorf("circuit breaker state should be HalfOpen")
		}
		cb.RecordResult(nil, time.Millisecond)
	}

	// state should transit to closed now
	if cb.State() != Closed {
		t.Errorf("circuit breaker state should be Closed")
	}
	if !cb.AcquirePermission() {
		t.Errorf("acquire permission should succeeded")
	}

	// insert 8 success results
	for i := 0; i < 8; i++ {
		if !cb.AcquirePermission() {
			t.Errorf("acquire permission should succeeded, i = %d", i)
		}
		cb.RecordResult(nil, time.Millisecond)
	}
	// insert 12 slow results
	for i := 0; i < 12; i++ {
		if !cb.AcquirePermission() {
			t.Errorf("acquire permission should succeeded, i = %d", i)
		}
		cb.RecordResult(nil, 11*time.Millisecond)
	}

	// state should transit to open now
	if cb.State() != Open {
		t.Errorf("circuit breaker state should be Open")
	}
	if cb.AcquirePermission() {
		t.Errorf("acquire permission should fail")
	}

	// wait 5 seconds for state to transit to half open
	now = now.Add(5 * time.Second)

	// insert 5 slow results
	for i := 0; i < 5; i++ {
		if !cb.AcquirePermission() {
			t.Errorf("acquire permission should succeeded, i = %d", i)
		}
		if cb.State() != HalfOpen {
			t.Errorf("circuit breaker state should be HalfOpen")
		}
		cb.RecordResult(nil, 11*time.Millisecond)
	}

	// state should transit to open now
	if cb.State() != Open {
		t.Errorf("circuit breaker state should be Open")
	}
	if cb.AcquirePermission() {
		t.Errorf("acquire permission should fail")
	}
}

func TestCountBased(t *testing.T) {
	policy := Policy{
		FailureRateThreshold:                  50,
		SlowCallRateThreshold:                 60,
		SlidingWindowType:                     CountBased,
		SlidingWindowSize:                     20,
		PermittedNumberOfCallsInHalfOpenState: 5,
		MinimumNumberOfCalls:                  10,
		SlowCallDurationThreshold:             time.Millisecond * 10,
		MaxWaitDurationInHalfOpenState:        5 * time.Second,
		WaitDurationInOpenState:               5 * time.Second,
	}

	cb := New(&policy)
	runSharedCases(t, cb)

	// transit to closed
	cb.SetState(Closed)
	// insert 12 success results
	for i := 0; i < 12; i++ {
		if !cb.AcquirePermission() {
			t.Errorf("acquire permission should succeeded, i = %d", i)
		}
		if cb.State() != Closed {
			t.Errorf("circuit breaker state should be Closed")
		}
		cb.RecordResult(nil, time.Millisecond)
	}
	// insert 10 failure results
	for i := 0; i < 10; i++ {
		if !cb.AcquirePermission() {
			t.Errorf("acquire permission should succeeded, i = %d", i)
		}
		if cb.State() != Closed {
			t.Errorf("circuit breaker state should be Closed")
		}
		cb.RecordResult(err, time.Millisecond)
	}
	// state should transit to open now
	if cb.State() != Open {
		t.Errorf("circuit breaker state should be Open")
	}
}

func TestTimeBased(t *testing.T) {
	policy := Policy{
		FailureRateThreshold:                  50,
		SlowCallRateThreshold:                 60,
		SlidingWindowType:                     TimeBased,
		SlidingWindowSize:                     20,
		PermittedNumberOfCallsInHalfOpenState: 5,
		MinimumNumberOfCalls:                  10,
		SlowCallDurationThreshold:             time.Millisecond * 10,
		MaxWaitDurationInHalfOpenState:        5 * time.Second,
		WaitDurationInOpenState:               5 * time.Second,
	}

	cb := New(&policy)
	runSharedCases(t, cb)
	// transit to closed

	cb.SetState(Closed)
	// insert 12 success results
	for i := 0; i < 12; i++ {
		if !cb.AcquirePermission() {
			t.Errorf("acquire permission should succeeded, i = %d", i)
		}
		if cb.State() != Closed {
			t.Errorf("circuit breaker state should be Closed")
		}
		cb.RecordResult(nil, time.Millisecond)
		now = now.Add(500 * time.Millisecond)
	}
	// insert 10 failure results
	for i := 0; i < 10; i++ {
		if !cb.AcquirePermission() {
			t.Errorf("acquire permission should succeeded, i = %d", i)
		}
		if cb.State() != Closed {
			t.Errorf("circuit breaker state should be Closed")
		}
		cb.RecordResult(err, time.Millisecond)
	}
	// state should be closed
	if cb.State() != Closed {
		t.Errorf("circuit breaker state should be Closed")
	}
	// evicts some success results
	now = now.Add(15500 * time.Millisecond)
	// add a new success result
	cb.RecordResult(nil, time.Millisecond)
	// state should be open now
	if cb.State() != Open {
		t.Errorf("circuit breaker state should be Open")
	}
}
