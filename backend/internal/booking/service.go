package booking

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/fksava45/reserve/internal/domain"
	"github.com/fksava45/reserve/internal/notification"
	"github.com/fksava45/reserve/internal/payment"
)

type Service struct {
	repo     Repository
	locks    LockManager
	payments *payment.Orchestrator
	notify   *notification.Service
	logger   *slog.Logger
	lockTTL  time.Duration
}

func NewService(repo Repository, locks LockManager, payments *payment.Orchestrator, notify *notification.Service, logger *slog.Logger) *Service {
	return &Service{repo: repo, locks: locks, payments: payments, notify: notify, logger: logger, lockTTL: 90 * time.Second}
}

func (s *Service) ListAvailability(ctx context.Context, from, to time.Time) ([]domain.Slot, error) {
	return s.repo.ListAvailability(ctx, from, to)
}

func (s *Service) Checkout(ctx context.Context, slotID string, customer domain.Customer) (domain.CheckoutSession, error) {
	slot, err := s.repo.GetSlot(ctx, slotID)
	if err != nil {
		return domain.CheckoutSession{}, err
	}
	if slot.Status != domain.SlotAvailable {
		return domain.CheckoutSession{}, ErrSlotUnavailable
	}
	session, err := s.payments.Checkout(ctx, slot, customer)
	if err != nil {
		return domain.CheckoutSession{}, err
	}
	s.logger.InfoContext(ctx, "checkout session created", "slot_id", slotID, "payment_intent_id", session.PaymentIntentID)
	return session, nil
}

func (s *Service) CreateBooking(ctx context.Context, slotID, paymentIntentID, googleBookingID string, customer domain.Customer) (domain.Booking, error) {
	if paymentIntentID == "" {
		return domain.Booking{}, ErrPaymentRequired
	}

	// Authorize payment first, per Google checkout expectations, before taking the Redis slot lock.
	authorized, err := s.payments.Authorize(ctx, paymentIntentID)
	if err != nil {
		return domain.Booking{}, fmt.Errorf("authorize payment: %w", err)
	}
	if authorized.Status != domain.PaymentAuthorized {
		return domain.Booking{}, ErrPaymentFailed
	}

	bookingID := newID("bk")
	lockKey := "slot:" + slotID
	locked, err := s.locks.Acquire(ctx, lockKey, bookingID, s.lockTTL)
	if err != nil {
		_ = s.payments.Void(ctx, paymentIntentID)
		return domain.Booking{}, err
	}
	if !locked {
		_ = s.payments.Void(ctx, paymentIntentID)
		return domain.Booking{}, ErrLockNotAcquired
	}
	defer s.locks.Release(ctx, lockKey, bookingID)

	now := time.Now().UTC()
	booking := domain.Booking{
		ID:              bookingID,
		GoogleBookingID: googleBookingID,
		SlotID:          slotID,
		Customer:        customer,
		Status:          domain.BookingPending,
		PaymentIntentID: paymentIntentID,
		PaymentStatus:   domain.PaymentAuthorized,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	created, err := s.repo.CreateBooking(ctx, booking)
	if err != nil {
		_ = s.payments.Void(ctx, paymentIntentID)
		return domain.Booking{}, err
	}

	if err := s.repo.MarkSlotBooked(ctx, slotID); err != nil {
		created.Status = domain.BookingFailed
		_ = s.repo.UpdateBooking(ctx, created)
		_ = s.payments.Void(ctx, paymentIntentID)
		return domain.Booking{}, err
	}

	captured, err := s.payments.Capture(ctx, paymentIntentID)
	if err != nil {
		created.Status = domain.BookingFailed
		_ = s.repo.UpdateBooking(ctx, created)
		_ = s.repo.MarkSlotAvailable(ctx, slotID)
		_ = s.payments.Void(ctx, paymentIntentID)
		return domain.Booking{}, fmt.Errorf("capture payment: %w", err)
	}

	created.Status = domain.BookingConfirmed
	created.PaymentStatus = captured.Status
	created.ConfirmationToken = newID("cnf")
	created.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateBooking(ctx, created); err != nil {
		return domain.Booking{}, err
	}
	if err := s.notify.BookingConfirmed(ctx, created); err != nil {
		s.logger.ErrorContext(ctx, "confirmation notification failed", "booking_id", created.ID, "error", err)
	}
	s.logger.InfoContext(ctx, "booking created", "booking_id", created.ID, "slot_id", slotID, "payment_intent_id", paymentIntentID)
	return created, nil
}

func (s *Service) GetBookingStatus(ctx context.Context, bookingID string) (domain.Booking, error) {
	return s.repo.GetBooking(ctx, bookingID)
}

func newID(prefix string) string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(buf)
}
