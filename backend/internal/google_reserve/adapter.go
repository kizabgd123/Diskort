package google_reserve

import (
	"context"
	"time"

	"diskort/fk-sava-45/backend/internal/booking"
)

type Adapter struct{ bookings *booking.Service }

func NewAdapter(bookings *booking.Service) *Adapter { return &Adapter{bookings: bookings} }

type ListAvailabilityRequest struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}
type ListAvailabilityResponse struct {
	Slots []booking.Slot `json:"slots"`
}

type CheckoutRequest struct {
	SlotID string `json:"slot_id"`
}
type CheckoutResponse struct {
	Session booking.CheckoutSession `json:"session"`
}

type CreateBookingRequest struct {
	SlotID          string           `json:"slot_id"`
	PaymentIntentID string           `json:"payment_intent_id"`
	Customer        booking.Customer `json:"customer"`
	GoogleBookingID string           `json:"google_booking_id"`
}
type CreateBookingResponse struct {
	BookingID string                `json:"booking_id"`
	Status    booking.BookingStatus `json:"status"`
}

type GetBookingStatusRequest struct {
	BookingID string `json:"booking_id"`
}
type GetBookingStatusResponse struct {
	Booking booking.Booking `json:"booking"`
}

func (a *Adapter) ListAvailability(ctx context.Context, req ListAvailabilityRequest) (ListAvailabilityResponse, error) {
	slots, err := a.bookings.ListAvailability(ctx, req.StartTime, req.EndTime)
	return ListAvailabilityResponse{Slots: slots}, err
}

func (a *Adapter) Checkout(ctx context.Context, req CheckoutRequest) (CheckoutResponse, error) {
	session, err := a.bookings.Checkout(ctx, req.SlotID)
	return CheckoutResponse{Session: session}, err
}

func (a *Adapter) CreateBooking(ctx context.Context, req CreateBookingRequest) (CreateBookingResponse, error) {
	created, err := a.bookings.CreateBooking(ctx, booking.CreateBookingInput{SlotID: req.SlotID, PaymentIntentID: req.PaymentIntentID, Customer: req.Customer, GoogleBookingID: req.GoogleBookingID})
	if err != nil {
		return CreateBookingResponse{}, err
	}
	return CreateBookingResponse{BookingID: created.ID, Status: created.Status}, nil
}

func (a *Adapter) GetBookingStatus(ctx context.Context, req GetBookingStatusRequest) (GetBookingStatusResponse, error) {
	found, err := a.bookings.GetBookingStatus(ctx, req.BookingID)
	return GetBookingStatusResponse{Booking: found}, err
}
