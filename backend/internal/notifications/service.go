package notifications

import (
	"context"

	"diskort/fk-sava-45/backend/internal/queue"
)

type Service struct{ queue queue.Publisher }

func NewService(queue queue.Publisher) *Service { return &Service{queue: queue} }

func (s *Service) BookingConfirmed(ctx context.Context, payload any) error {
	return s.queue.Publish(ctx, queue.Job{Type: "notification.booking_confirmed", Payload: payload})
}
