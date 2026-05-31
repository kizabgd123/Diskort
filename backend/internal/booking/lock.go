package booking

import (
	"context"
	"sync"
	"time"
)

type lockRecord struct {
	owner     string
	expiresAt time.Time
}

type MemoryLockManager struct {
	mu    sync.Mutex
	locks map[string]lockRecord
}

func NewMemoryLockManager() *MemoryLockManager {
	return &MemoryLockManager{locks: map[string]lockRecord{}}
}

func (m *MemoryLockManager) Acquire(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	if current, ok := m.locks[key]; ok && current.expiresAt.After(now) && current.owner != owner {
		return false, nil
	}
	m.locks[key] = lockRecord{owner: owner, expiresAt: now.Add(ttl)}
	return true, nil
}

func (m *MemoryLockManager) Release(ctx context.Context, key, owner string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if current, ok := m.locks[key]; ok && current.owner == owner {
		delete(m.locks, key)
	}
	return nil
}
