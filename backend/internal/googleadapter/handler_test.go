package googleadapter

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fksava45/reserve/internal/booking"
	"github.com/fksava45/reserve/internal/domain"
	"github.com/fksava45/reserve/internal/notification"
	"github.com/fksava45/reserve/internal/payment"
	"github.com/fksava45/reserve/internal/queue"
)

func TestEndToEndCheckoutCreateBookingAndStatus(t *testing.T) {
	server, slots := testServer(t)
	customer := domain.Customer{Name: "Milan", Email: "milan@example.com", Phone: "+38160123456"}

	checkoutBody := postJSON(t, server, "/google-reserve/checkout", checkoutRequest{SlotID: slots[0].ID, Customer: customer}, http.StatusOK)
	var session domain.CheckoutSession
	decode(t, checkoutBody, &session)
	if session.PaymentIntentID == "" || session.AmountRSD != slots[0].PriceRSD {
		t.Fatalf("unexpected checkout session: %+v", session)
	}

	createBody := postJSON(t, server, "/google-reserve/bookings", createBookingRequest{SlotID: slots[0].ID, PaymentIntentID: session.PaymentIntentID, GoogleBookingID: "google-1", Customer: customer}, http.StatusCreated)
	var created domain.Booking
	decode(t, createBody, &created)
	if created.Status != domain.BookingConfirmed || created.PaymentStatus != domain.PaymentCaptured {
		t.Fatalf("booking was not confirmed and captured: %+v", created)
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/google-reserve/bookings/"+created.ID, nil)
	statusRes := httptest.NewRecorder()
	server.ServeHTTP(statusRes, statusReq)
	if statusRes.Code != http.StatusOK {
		t.Fatalf("status code = %d body=%s", statusRes.Code, statusRes.Body.String())
	}

	postJSON(t, server, "/google-reserve/bookings", createBookingRequest{SlotID: slots[0].ID, PaymentIntentID: session.PaymentIntentID, GoogleBookingID: "google-duplicate", Customer: customer}, http.StatusConflict)
}

func TestAvailabilityEndpoint(t *testing.T) {
	server, slots := testServer(t)
	from := slots[0].StartAt.Add(-time.Minute).Format(time.RFC3339)
	to := slots[len(slots)-1].EndAt.Add(time.Minute).Format(time.RFC3339)
	req := httptest.NewRequest(http.MethodGet, "/google-reserve/availability?from="+from+"&to="+to, nil)
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status code = %d body=%s", res.Code, res.Body.String())
	}
	var payload struct {
		Availability []domain.Slot `json:"availability"`
	}
	decode(t, res.Body.Bytes(), &payload)
	if len(payload.Availability) == 0 {
		t.Fatal("expected availability")
	}
}

func testServer(t *testing.T) (http.Handler, []domain.Slot) {
	t.Helper()
	slots := booking.SeedSlots(time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC))
	repo := booking.NewMemoryRepository(slots)
	locks := booking.NewMemoryLockManager()
	publisher := &queue.MemoryPublisher{}
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	notifications := notification.NewService(publisher, logger)
	payments := payment.NewOrchestrator(payment.NewFakeStripe(), payment.FakeSCA{})
	service := booking.NewService(repo, locks, payments, notifications, logger)
	mux := http.NewServeMux()
	NewHandler(service, logger).Register(mux)
	return mux, slots
}

func postJSON(t *testing.T, handler http.Handler, path string, body any, expectedStatus int) []byte {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != expectedStatus {
		t.Fatalf("%s status code = %d want %d body=%s", path, res.Code, expectedStatus, res.Body.String())
	}
	return res.Body.Bytes()
}

func decode(t *testing.T, body []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("decode %s: %v", string(body), err)
	}
}
