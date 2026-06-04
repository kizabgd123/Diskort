package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/fksava45/reserve/internal/booking"
	"github.com/fksava45/reserve/internal/googleadapter"
	"github.com/fksava45/reserve/internal/monitoring"
	"github.com/fksava45/reserve/internal/notification"
	"github.com/fksava45/reserve/internal/payment"
	"github.com/fksava45/reserve/internal/queue"
)

func main() {
	logger := monitoring.NewLogger("fk-sava45-reserve")
	repo := booking.NewMemoryRepository(booking.SeedSlots(time.Now().UTC()))

	var locks booking.LockManager = booking.NewMemoryLockManager()
	if os.Getenv("REDIS_URL") != "" {
		locks = booking.NewRedisLockManagerFromEnv()
	}

	var stripe payment.StripeClient = payment.NewFakeStripe()
	if os.Getenv("STRIPE_SECRET_KEY") != "" {
		stripe = payment.NewStripeHTTPClientFromEnv()
	}

	publisher := &queue.MemoryPublisher{}
	notifications := notification.NewService(publisher, logger)
	payments := payment.NewOrchestrator(stripe, payment.FakeSCA{})
	bookingService := booking.NewService(repo, locks, payments, notifications, logger)

	mux := http.NewServeMux()
	googleadapter.NewHandler(bookingService, logger).Register(mux)

	addr := ":8080"
	if value := os.Getenv("HTTP_ADDR"); value != "" {
		addr = value
	}
	logger.Info("starting reserve backend", slog.String("addr", addr))
	if err := http.ListenAndServe(addr, requestLogger(logger, mux)); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		logger.InfoContext(r.Context(), "request", "method", r.Method, "path", r.URL.Path, "duration_ms", time.Since(started).Milliseconds())
	})
}
