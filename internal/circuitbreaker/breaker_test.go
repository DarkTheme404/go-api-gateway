package circuitbreaker

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_ClosedToOpen(t *testing.T) {
	cb := New(3, time.Second)

	for i := 0; i < 3; i++ {
		err := cb.Execute(func() error { return errors.New("fail") })
		if err == nil {
			t.Fatal("expected error")
		}
	}

	if cb.State() != StateOpen {
		t.Fatalf("expected open, got %s", cb.State())
	}

	err := cb.Execute(func() error { return nil })
	if err != ErrCircuitOpen {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreaker_OpenToHalfOpen(t *testing.T) {
	cb := New(2, 50*time.Millisecond)

	cb.Execute(func() error { return errors.New("fail") })
	cb.Execute(func() error { return errors.New("fail") })

	if cb.State() != StateOpen {
		t.Fatalf("expected open, got %s", cb.State())
	}

	time.Sleep(60 * time.Millisecond)

	err := cb.Execute(func() error { return nil })
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}

	if cb.State() != StateClosed {
		t.Fatalf("expected closed, got %s", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenFailReopens(t *testing.T) {
	cb := New(2, 50*time.Millisecond)

	cb.Execute(func() error { return errors.New("fail") })
	cb.Execute(func() error { return errors.New("fail") })

	time.Sleep(60 * time.Millisecond)

	err := cb.Execute(func() error { return errors.New("fail again") })
	if err == nil {
		t.Fatal("expected error")
	}

	if cb.State() != StateOpen {
		t.Fatalf("expected open, got %s", cb.State())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := New(2, time.Second)

	cb.Execute(func() error { return errors.New("fail") })
	cb.Execute(func() error { return errors.New("fail") })

	if cb.State() != StateOpen {
		t.Fatalf("expected open, got %s", cb.State())
	}

	cb.Reset()

	if cb.State() != StateClosed {
		t.Fatalf("expected closed after reset, got %s", cb.State())
	}
}

func TestCircuitBreaker_SuccessResetsCount(t *testing.T) {
	cb := New(3, time.Second)

	cb.Execute(func() error { return errors.New("fail") })
	cb.Execute(func() error { return errors.New("fail") })

	if cb.FailureCount() != 2 {
		t.Fatalf("expected 2 failures, got %d", cb.FailureCount())
	}

	cb.Execute(func() error { return nil })

	if cb.FailureCount() != 0 {
		t.Fatalf("expected 0 failures after success, got %d", cb.FailureCount())
	}
}
