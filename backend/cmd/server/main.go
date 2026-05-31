package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"diskort/fk-sava-45/backend/internal/booking"
	reserve "diskort/fk-sava-45/backend/internal/google_reserve"
	"diskort/fk-sava-45/backend/internal/httpapi"
	"diskort/fk-sava-45/backend/internal/locking"
	"diskort/fk-sava-45/backend/internal/monitoring"
	"diskort/fk-sava-45/backend/internal/notifications"
	"diskort/fk-sava-45/backend/internal/payments"
	"diskort/fk-sava-45/backend/internal/queue"
)

func main() {
	store := booking.NewInMemoryStore(booking.DefaultSlots(time.Now()))
	locker := locking.Locker(locking.NewInMemoryLocker())
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		redisLocker, err := locking.NewRedisLocker(redisURL)
		if err != nil {
			log.Fatalf("invalid REDIS_URL: %v", err)
		}
		locker = redisLocker
	}

	psp := payments.PSP(payments.FakeStripe{})
	if stripeSecret := os.Getenv("STRIPE_SECRET_KEY"); stripeSecret != "" {
		psp = payments.StripeClient{SecretKey: stripeSecret}
	}
	orchestrator := payments.NewOrchestrator(psp, payments.ThreeDS{BaseURL: os.Getenv("THREEDS_BASE_URL")})
	jobs := &queue.InMemoryQueue{}
	notifier := notifications.NewService(jobs)
	logger := monitoring.NewLogger()
	bookings := booking.NewService(store, locker, orchestrator, notifier, logger)
	adapter := reserve.NewAdapter(bookings)
	server := httpapi.NewServer(adapter)

	addr := os.Getenv("RESERVE_HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("FK Sava 45 Google Reserve backend listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, server.Handler()))
}
