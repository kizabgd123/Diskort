package booking_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"diskort/fk-sava-45/backend/internal/booking"
	"diskort/fk-sava-45/backend/internal/locking"
	"diskort/fk-sava-45/backend/internal/monitoring"
	"diskort/fk-sava-45/backend/internal/notifications"
	"diskort/fk-sava-45/backend/internal/payments"
	"diskort/fk-sava-45/backend/internal/queue"
)

func newService() (*booking.Service, booking.Slot) {
	slots := booking.DefaultSlots(time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC))
	store := booking.NewInMemoryStore(slots)
	orchestrator := payments.NewOrchestrator(payments.FakeStripe{}, payments.ThreeDS{})
	service := booking.NewService(store, locking.NewInMemoryLocker(), orchestrator, notifications.NewService(&queue.InMemoryQueue{}), monitoring.NewLogger())
	return service, slots[0]
}

func TestCheckoutAndCreateBookingCapturesPayment(t *testing.T) {
	ctx := context.Background()
	service, slot := newService()

	checkout, err := service.Checkout(ctx, slot.ID)
	if err != nil {
		t.Fatalf("checkout failed: %v", err)
	}
	if checkout.PaymentIntentID == "" {
		t.Fatal("expected payment intent")
	}

	created, err := service.CreateBooking(ctx, booking.CreateBookingInput{SlotID: slot.ID, PaymentIntentID: checkout.PaymentIntentID, Customer: booking.Customer{Name: "Test Player", Phone: "+38160111222", Email: "test@example.com"}, GoogleBookingID: "google-1"})
	if err != nil {
		t.Fatalf("create booking failed: %v", err)
	}
	if created.Status != booking.BookingConfirmed {
		t.Fatalf("expected confirmed booking, got %s", created.Status)
	}
	if created.PaymentCaptureID == "" {
		t.Fatal("expected captured payment")
	}

	if _, err := service.Checkout(ctx, slot.ID); err == nil {
		t.Fatal("expected booked slot to become unavailable")
	}
}

func TestConcurrentCreateBookingAllowsSingleWinner(t *testing.T) {
	ctx := context.Background()
	service, slot := newService()
	checkout, err := service.Checkout(ctx, slot.ID)
	if err != nil {
		t.Fatalf("checkout failed: %v", err)
	}

	var wg sync.WaitGroup
	wins := 0
	var mu sync.Mutex
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := service.CreateBooking(ctx, booking.CreateBookingInput{SlotID: slot.ID, PaymentIntentID: checkout.PaymentIntentID, Customer: booking.Customer{Name: "Team", Phone: "+381", Email: "team@example.com"}})
			if err == nil {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if wins != 1 {
		t.Fatalf("expected exactly one booking winner, got %d", wins)
	}
}
