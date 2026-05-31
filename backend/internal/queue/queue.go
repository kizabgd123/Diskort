package queue

import (
	"context"
	"sync"
)

type Job struct {
	Type    string
	Payload any
}

type Publisher interface {
	Publish(ctx context.Context, job Job) error
}

type InMemoryQueue struct {
	mu   sync.Mutex
	Jobs []Job
}

func (q *InMemoryQueue) Publish(ctx context.Context, job Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.Jobs = append(q.Jobs, job)
	return nil
}
