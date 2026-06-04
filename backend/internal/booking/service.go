package booking

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"diskort/fk-sava-45/backend/internal/locking"
	"diskort/fk-sava-45/backend/internal/monitoring"
	"diskort/fk-sava-45/backend/internal/notifications"
	"diskort/fk-sava-45/backend/internal/payments"
)

type Service struct {
	store         Store
	locker        locking.Locker
	payments      *payments.Orchestrator
	notifications *notifications.Service
	logger        monitoring.Logger
	lockTTL       time.Duration
}

func NewService(store Store, locker locking.Locker, orchestrator *payments.Orchestrator, notifications *notifications.Service, logger monitoring.Logger) *Service {
	return &Service{store: store, locker: locker, payments: orchestrator, notifications: notifications, logger: logger, lockTTL: 90 * time.Second}
}

func (s *Service) ListAvailability(ctx context.Context, from, to time.Time) ([]Slot, error) {
	return s.store.ListAvailability(ctx, from, to)
}

func (s *Service) Checkout(ctx context.Context, slotID string) (CheckoutSession, error) {
	slot, err := s.store.GetSlot(ctx, slotID)
	if err != nil {
		return CheckoutSession{}, err
	}
	intent, challenge, err := s.payments.PrepareCheckout(ctx, slot.PriceMinor, slot.Currency, "checkout_"+slot.ID)
	if err != nil {
		return CheckoutSession{}, err
	}
	return CheckoutSession{Slot: slot, PaymentIntentID: intent.ID, ClientSecret: intent.ClientSecret, Requires3DS: intent.Requires3DS, ThreeDSChallenge: challenge}, nil
}

type CreateBookingInput struct {
	SlotID          string
	PaymentIntentID string
	Customer        Customer
	GoogleBookingID string
}

func (s *Service) CreateBooking(ctx context.Context, input CreateBookingInput) (Booking, error) {
	if input.PaymentIntentID == "" {
		return Booking{}, ErrPaymentNotReady
	}
	auth, err := s.payments.Authorize(ctx, input.PaymentIntentID)
	if err != nil {
		return Booking{}, err
	}

	lockToken := newID("lock")
	lockKey := "slot:" + input.SlotID
	if err := s.locker.Acquire(ctx, lockKey, lockToken, s.lockTTL); err != nil {
		_ = s.payments.Rollback(ctx, input.PaymentIntentID)
		return Booking{}, err
	}
	defer s.locker.Release(context.WithoutCancel(ctx), lockKey, lockToken)

	slot, err := s.store.GetSlot(ctx, input.SlotID)
	if err != nil {
		_ = s.payments.Rollback(ctx, input.PaymentIntentID)
		return Booking{}, err
	}
	if err := s.store.MarkSlotBooked(ctx, input.SlotID); err != nil {
		_ = s.payments.Rollback(ctx, input.PaymentIntentID)
		return Booking{}, err
	}

	capture, err := s.payments.Capture(ctx, auth.ID)
	if err != nil {
		_ = s.payments.Rollback(ctx, input.PaymentIntentID)
		return Booking{}, err
	}

	now := time.Now().UTC()
	booking := Booking{ID: newID("book"), GoogleBookingID: input.GoogleBookingID, SlotID: input.SlotID, Customer: input.Customer, Status: BookingConfirmed, PaymentIntentID: input.PaymentIntentID, PaymentAuthorizationID: auth.ID, PaymentCaptureID: capture.ID, AmountMinor: slot.PriceMinor, Currency: slot.Currency, CreatedAt: now, UpdatedAt: now}
	created, err := s.store.CreateBooking(ctx, booking)
	if err != nil {
		_ = s.payments.Rollback(ctx, input.PaymentIntentID)
		return Booking{}, err
	}
	if err := s.notifications.BookingConfirmed(ctx, created); err != nil {
		s.logger.BookingEvent(ctx, "notification_enqueue_failed", "booking_id", created.ID, "error", err.Error())
	}
	s.logger.BookingEvent(ctx, "booking_confirmed", "booking_id", created.ID, "slot_id", created.SlotID)
	return created, nil
}

func (s *Service) GetBookingStatus(ctx context.Context, bookingID string) (Booking, error) {
	return s.store.GetBooking(ctx, bookingID)
}

func newID(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}

func IsConflict(err error) bool {
	return errors.Is(err, ErrSlotUnavailable) || errors.Is(err, locking.ErrLockHeld)
}
