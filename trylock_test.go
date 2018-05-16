package trylock

import (
	"testing"
	"time"
)

func TestMutexLockTryLock(t *testing.T) {
	mu := New()

	if ok := mu.TryLock(0); !ok {
		t.Errorf("cannot Lock !!!")
	}
	if ok := mu.TryLock(0); ok {
		t.Errorf("cannot Lock twice !!!")
	}

	mu.Unlock()
}

func TestMutexLockTryLockTimeout(t *testing.T) {
	mu := New()
	mu.Lock()

	if ok := mu.TryLock(10 * time.Millisecond); ok {
		t.Errorf("should not Lock in 10ms !!!")
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		mu.Unlock()
	}()
	if ok := mu.TryLock(200 * time.Millisecond); !ok {
		t.Errorf("cannot Lock after 200ms !!!")
	}

	mu.Unlock()
}

func TestMutexLockUnLockTwice(t *testing.T) {
	mu := New()
	mu.Lock()
	defer func() {
		if x := recover(); x != nil {
			if x != "Unlock() failed" {
				t.Errorf("unexpect panic")
			}
		} else {
			t.Errorf("should panic after unlock twice")
		}
	}()
	mu.Unlock()
	mu.Unlock()
}

func TestMutexLockRLockTwice(t *testing.T) {
	mu := New()
	mu.RLock()
	mu.RLock()
	mu.RUnlock()
	mu.RUnlock()
}

func TestMutexLockUnLockInvalid(t *testing.T) {
	mu := New()
	mu.Lock()
	defer func() {
		if x := recover(); x != nil {
			if x != "RUnlock() failed" {
				t.Errorf("unexpect panic")
			}
		} else {
			t.Errorf("should panic after RUnlock a write lock")
		}
	}()
	mu.RUnlock()
}
