package trylock

import (
	"sync"
	"sync/atomic"
	"time"
)

// MutexLock is a simple sync.RWMutex + ability to try to Lock.
type MutexLock struct {
	// if v == 0, no lock
	// if v == -1, write lock
	// if v > 0, read lock, and v is the number of readers
	v *int32

	// broadcast channel
	ch chan struct{}
	// broadcast channel locker
	chLock sync.Mutex
}

// confirm MutexLock implements sync.Locker
var _ sync.Locker = &MutexLock{}

// New returns a new MutexLock
func New() *MutexLock {
	v := int32(0)
	ch := make(chan struct{}, 1)
	return &MutexLock{v: &v, ch: ch}
}

// TryLock tries to lock for writing. It returns true in case of success, false if timeout.
// A negative timeout means no timeout. If timeout is 0 that means try at once and quick return.
// If the lock is currently held by another goroutine, TryLock will wait until it has a chance to acquire it.
func (m *MutexLock) TryLock(timeout time.Duration) bool {
	// deadline for timeout
	deadline := time.Now().Add(timeout)

	for {
		if atomic.CompareAndSwapInt32(m.v, 0, -1) {
			return true
		}

		// get broadcast channel
		m.chLock.Lock()
		ch := m.ch
		m.chLock.Unlock()

		// Waiting for wake up before trying again.
		if timeout < 0 {
			// waitting
			<-ch
		} else {
			elapsed := time.Until(deadline)
			if elapsed <= 0 {
				// timeout
				return false
			}

			select {
			case <-ch:
				// wake up to try again
			case <-time.After(elapsed):
				// timeout
				return false
			}
		}
	}
}

// RTryLock tries to lock for reading. It returns true in case of success, false if timeout.
// A negative timeout means no timeout. If timeout is 0 that means try at once and quick return.
func (m *MutexLock) RTryLock(timeout time.Duration) bool {
	// deadline for timeout
	deadline := time.Now().Add(timeout)

	for {
		n := atomic.LoadInt32(m.v)
		if n >= 0 {
			if atomic.CompareAndSwapInt32(m.v, n, n+1) {
				return true
			}
		}

		// get broadcast channel
		m.chLock.Lock()
		ch := m.ch
		m.chLock.Unlock()

		// Waiting for wake up before trying again.
		if timeout < 0 {
			// waitting
			<-ch
		} else {
			elapsed := time.Until(deadline)
			if elapsed <= 0 {
				// timeout
				return false
			}

			select {
			case <-ch:
				// wake up to try again
			case <-time.After(elapsed):
				// timeout
				return false
			}
		}
	}
}

// Lock locks for writing. If the lock is already locked for reading or writing, Lock blocks until the lock is available.
func (m *MutexLock) Lock() {
	m.TryLock(-1)
}

// RLock locks for reading. If the lock is already locked for writing, RLock blocks until the lock is available.
func (m *MutexLock) RLock() {
	m.RTryLock(-1)
}

// Unlock unlocks for writing. It is a panic if m is not locked for writing on entry to Unlock.
func (m *MutexLock) Unlock() {
	if ok := atomic.CompareAndSwapInt32(m.v, -1, 0); !ok {
		panic("Unlock() failed")
	}

	m.broadcast()
}

// RUnlock unlocks for reading. It is a panic if m is not locked for reading on entry to Unlock.
func (m *MutexLock) RUnlock() {
	n := atomic.AddInt32(m.v, -1)
	if n < 0 {
		panic("RUnlock() failed")
	}

	if n == 0 {
		m.broadcast()
	}
}

func (m *MutexLock) broadcast() {
	newCh := make(chan struct{}, 1)

	m.chLock.Lock()
	ch := m.ch
	m.ch = newCh
	m.chLock.Unlock()

	// send broadcast signal
	close(ch)
}
