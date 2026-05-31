package notification

import (
	"context"
	"log/slog"

	"github.com/fksava45/reserve/internal/domain"
	"github.com/fksava45/reserve/internal/queue"
)

type Service struct {
	publisher queue.Publisher
	logger    *slog.Logger
}

func NewService(publisher queue.Publisher, logger *slog.Logger) *Service {
	return &Service{publisher: publisher, logger: logger}
}

func (s *Service) BookingConfirmed(ctx context.Context, booking domain.Booking) error {
	s.logger.InfoContext(ctx, "booking confirmation queued", "booking_id", booking.ID, "slot_id", booking.SlotID)
	return s.publisher.Publish(ctx, queue.Job{Name: "booking.confirmed", Payload: map[string]string{"booking_id": booking.ID, "slot_id": booking.SlotID}})
}
