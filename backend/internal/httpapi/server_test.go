package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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

func TestGoogleReserveHTTPWorkflow(t *testing.T) {
	slots := booking.DefaultSlots(time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC))
	store := booking.NewInMemoryStore(slots)
	bookings := booking.NewService(store, locking.NewInMemoryLocker(), payments.NewOrchestrator(payments.FakeStripe{}, payments.ThreeDS{}), notifications.NewService(&queue.InMemoryQueue{}), monitoring.NewLogger())
	server := httptest.NewServer(httpapi.NewServer(reserve.NewAdapter(bookings)).Handler())
	defer server.Close()

	checkoutResp := postJSON[reserve.CheckoutResponse](t, server.URL+"/v1/google-reserve/Checkout", reserve.CheckoutRequest{SlotID: slots[0].ID})
	if checkoutResp.Session.PaymentIntentID == "" {
		t.Fatal("missing payment intent")
	}

	createResp := postJSON[reserve.CreateBookingResponse](t, server.URL+"/v1/google-reserve/CreateBooking", reserve.CreateBookingRequest{SlotID: slots[0].ID, PaymentIntentID: checkoutResp.Session.PaymentIntentID, Customer: booking.Customer{Name: "Ana", Phone: "+381", Email: "ana@example.com"}})
	if createResp.BookingID == "" {
		t.Fatal("missing booking id")
	}

	statusResp := postJSON[reserve.GetBookingStatusResponse](t, server.URL+"/v1/google-reserve/GetBookingStatus", reserve.GetBookingStatusRequest{BookingID: createResp.BookingID})
	if statusResp.Booking.Status != booking.BookingConfirmed {
		t.Fatalf("status = %s", statusResp.Booking.Status)
	}
}

func postJSON[T any](t *testing.T, url string, payload any) T {
	t.Helper()
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("content-type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status %d", resp.StatusCode)
	}
	var out T
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	return out
}
