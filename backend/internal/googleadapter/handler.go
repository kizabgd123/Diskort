package googleadapter

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/fksava45/reserve/internal/booking"
	"github.com/fksava45/reserve/internal/domain"
)

type Handler struct {
	bookings *booking.Service
	logger   *slog.Logger
}

func NewHandler(bookings *booking.Service, logger *slog.Logger) *Handler {
	return &Handler{bookings: bookings, logger: logger}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("GET /google-reserve/availability", h.listAvailability)
	mux.HandleFunc("POST /google-reserve/checkout", h.checkout)
	mux.HandleFunc("POST /google-reserve/bookings", h.createBooking)
	mux.HandleFunc("GET /google-reserve/bookings/{booking_id}", h.getBookingStatus)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "fk-sava45-google-reserve"})
}

func (h *Handler) listAvailability(w http.ResponseWriter, r *http.Request) {
	from, to, err := availabilityWindow(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	slots, err := h.bookings.ListAvailability(r.Context(), from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"availability": slots})
}

type checkoutRequest struct {
	SlotID   string          `json:"slot_id"`
	Customer domain.Customer `json:"customer"`
}

func (h *Handler) checkout(w http.ResponseWriter, r *http.Request) {
	var req checkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.SlotID == "" {
		writeError(w, http.StatusBadRequest, errors.New("slot_id is required"))
		return
	}
	session, err := h.bookings.Checkout(r.Context(), req.SlotID, req.Customer)
	if err != nil {
		writeBookingError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, session)
}

type createBookingRequest struct {
	SlotID          string          `json:"slot_id"`
	PaymentIntentID string          `json:"payment_intent_id"`
	GoogleBookingID string          `json:"google_booking_id"`
	Customer        domain.Customer `json:"customer"`
}

func (h *Handler) createBooking(w http.ResponseWriter, r *http.Request) {
	var req createBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	created, err := h.bookings.CreateBooking(r.Context(), req.SlotID, req.PaymentIntentID, req.GoogleBookingID, req.Customer)
	if err != nil {
		writeBookingError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) getBookingStatus(w http.ResponseWriter, r *http.Request) {
	bookingID := r.PathValue("booking_id")
	item, err := h.bookings.GetBookingStatus(r.Context(), bookingID)
	if err != nil {
		writeBookingError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func availabilityWindow(r *http.Request) (time.Time, time.Time, error) {
	fromText := r.URL.Query().Get("from")
	toText := r.URL.Query().Get("to")
	if fromText == "" {
		fromText = time.Now().UTC().Format(time.RFC3339)
	}
	from, err := time.Parse(time.RFC3339, fromText)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if toText == "" {
		return from, from.Add(7 * 24 * time.Hour), nil
	}
	to, err := time.Parse(time.RFC3339, toText)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return from, to, nil
}

func writeBookingError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, booking.ErrSlotNotFound), errors.Is(err, booking.ErrBookingNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, booking.ErrSlotUnavailable), errors.Is(err, booking.ErrDuplicateBooking), errors.Is(err, booking.ErrLockNotAcquired):
		writeError(w, http.StatusConflict, err)
	case errors.Is(err, booking.ErrPaymentRequired), errors.Is(err, booking.ErrPaymentFailed):
		writeError(w, http.StatusPaymentRequired, err)
	default:
		writeError(w, http.StatusInternalServerError, err)
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{"error": err.Error(), "status": status})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
