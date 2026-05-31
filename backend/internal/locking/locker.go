package locking

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrLockHeld = errors.New("slot lock already held")

type Locker interface {
	Acquire(ctx context.Context, key, token string, ttl time.Duration) error
	Release(ctx context.Context, key, token string) error
}

type lockEntry struct {
	token     string
	expiresAt time.Time
}

type InMemoryLocker struct {
	mu    sync.Mutex
	locks map[string]lockEntry
}

func NewInMemoryLocker() *InMemoryLocker { return &InMemoryLocker{locks: map[string]lockEntry{}} }

func (l *InMemoryLocker) Acquire(ctx context.Context, key, token string, ttl time.Duration) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now().UTC()
	if entry, ok := l.locks[key]; ok && entry.expiresAt.After(now) {
		return fmt.Errorf("%w: %s", ErrLockHeld, key)
	}
	l.locks[key] = lockEntry{token: token, expiresAt: now.Add(ttl)}
	return nil
}

func (l *InMemoryLocker) Release(ctx context.Context, key, token string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if entry, ok := l.locks[key]; ok && entry.token == token {
		delete(l.locks, key)
	}
	return nil
}
