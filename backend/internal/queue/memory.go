package queue

import (
	"context"
	"sync"
)

type Job struct {
	Name    string
	Payload map[string]string
}

type Publisher interface {
	Publish(ctx context.Context, job Job) error
}

type MemoryPublisher struct {
	mu   sync.Mutex
	Jobs []Job
}

func (p *MemoryPublisher) Publish(ctx context.Context, job Job) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Jobs = append(p.Jobs, job)
	return nil
}
