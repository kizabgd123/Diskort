package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"diskort/fk-sava-45/backend/internal/booking"
	reserve "diskort/fk-sava-45/backend/internal/google_reserve"
)

type Server struct {
	adapter *reserve.Adapter
	mux     *http.ServeMux
}

func NewServer(adapter *reserve.Adapter) *Server {
	s := &Server{adapter: adapter, mux: http.NewServeMux()}
	s.mux.HandleFunc("/healthz", s.health)
	s.mux.HandleFunc("POST /v1/google-reserve/ListAvailability", s.listAvailability)
	s.mux.HandleFunc("POST /v1/google-reserve/Checkout", s.checkout)
	s.mux.HandleFunc("POST /v1/google-reserve/CreateBooking", s.createBooking)
	s.mux.HandleFunc("POST /v1/google-reserve/GetBookingStatus", s.getBookingStatus)
	return s
}

func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) listAvailability(w http.ResponseWriter, r *http.Request) {
	var req reserve.ListAvailabilityRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.StartTime.IsZero() {
		req.StartTime = time.Now().UTC()
	}
	if req.EndTime.IsZero() {
		req.EndTime = req.StartTime.Add(24 * time.Hour)
	}
	resp, err := s.adapter.ListAvailability(r.Context(), req)
	writeResult(w, resp, err)
}

func (s *Server) checkout(w http.ResponseWriter, r *http.Request) {
	var req reserve.CheckoutRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	resp, err := s.adapter.Checkout(r.Context(), req)
	writeResult(w, resp, err)
}

func (s *Server) createBooking(w http.ResponseWriter, r *http.Request) {
	var req reserve.CreateBookingRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	resp, err := s.adapter.CreateBooking(r.Context(), req)
	writeResult(w, resp, err)
}

func (s *Server) getBookingStatus(w http.ResponseWriter, r *http.Request) {
	var req reserve.GetBookingStatusRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	resp, err := s.adapter.GetBookingStatus(r.Context(), req)
	writeResult(w, resp, err)
}

func readJSON(r *http.Request, out any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(out)
}

func writeResult(w http.ResponseWriter, payload any, err error) {
	if err == nil {
		writeJSON(w, http.StatusOK, payload)
		return
	}
	status := http.StatusInternalServerError
	if errors.Is(err, booking.ErrSlotNotFound) || errors.Is(err, booking.ErrBookingNotFound) {
		status = http.StatusNotFound
	}
	if booking.IsConflict(err) {
		status = http.StatusConflict
	}
	if errors.Is(err, booking.ErrPaymentNotReady) {
		status = http.StatusBadRequest
	}
	writeError(w, status, err)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
