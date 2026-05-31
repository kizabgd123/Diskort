package booking

import (
	"bytes"
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/fksava45/reserve/internal/domain"
	"github.com/fksava45/reserve/internal/notification"
	"github.com/fksava45/reserve/internal/payment"
	"github.com/fksava45/reserve/internal/queue"
)

func TestConcurrentCreateBookingAllowsOnlyOneConfirmation(t *testing.T) {
	ctx := context.Background()
	slots := SeedSlots(time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC))
	service := newTestService(slots)
	customer := domain.Customer{Name: "Ana", Email: "ana@example.com", Phone: "+38160111111"}
	checkoutA, err := service.Checkout(ctx, slots[0].ID, customer)
	if err != nil {
		t.Fatal(err)
	}
	checkoutB, err := service.Checkout(ctx, slots[0].ID, customer)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, intentID := range []string{checkoutA.PaymentIntentID, checkoutB.PaymentIntentID} {
		wg.Add(1)
		go func(paymentIntentID string) {
			defer wg.Done()
			_, err := service.CreateBooking(ctx, slots[0].ID, paymentIntentID, "", customer)
			results <- err
		}(intentID)
	}
	wg.Wait()
	close(results)

	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("successes = %d, want exactly one", successes)
	}
}

func newTestService(slots []domain.Slot) *Service {
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	repo := NewMemoryRepository(slots)
	locks := NewMemoryLockManager()
	notifications := notification.NewService(&queue.MemoryPublisher{}, logger)
	payments := payment.NewOrchestrator(payment.NewFakeStripe(), payment.FakeSCA{})
	return NewService(repo, locks, payments, notifications, logger)
}
