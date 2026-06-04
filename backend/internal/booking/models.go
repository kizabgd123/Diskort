package booking

import "time"

type SlotStatus string

const (
	SlotAvailable SlotStatus = "available"
	SlotLocked    SlotStatus = "locked"
	SlotBooked    SlotStatus = "booked"
	SlotCancelled SlotStatus = "cancelled"
)

type BookingStatus string

const (
	BookingPendingPayment BookingStatus = "pending_payment"
	BookingConfirmed      BookingStatus = "confirmed"
	BookingCancelled      BookingStatus = "cancelled"
	BookingFailed         BookingStatus = "failed"
)

type Slot struct {
	ID         string     `json:"id"`
	CourtID    string     `json:"court_id"`
	CourtName  string     `json:"court_name"`
	Sport      string     `json:"sport"`
	StartsAt   time.Time  `json:"starts_at"`
	EndsAt     time.Time  `json:"ends_at"`
	PriceMinor int64      `json:"price_minor"`
	Currency   string     `json:"currency"`
	Status     SlotStatus `json:"status"`
}

type Customer struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Email string `json:"email"`
}

type Booking struct {
	ID                     string        `json:"id"`
	GoogleBookingID        string        `json:"google_booking_id,omitempty"`
	SlotID                 string        `json:"slot_id"`
	Customer               Customer      `json:"customer"`
	Status                 BookingStatus `json:"status"`
	PaymentIntentID        string        `json:"payment_intent_id"`
	PaymentAuthorizationID string        `json:"payment_authorization_id"`
	PaymentCaptureID       string        `json:"payment_capture_id,omitempty"`
	AmountMinor            int64         `json:"amount_minor"`
	Currency               string        `json:"currency"`
	CreatedAt              time.Time     `json:"created_at"`
	UpdatedAt              time.Time     `json:"updated_at"`
}

type CheckoutSession struct {
	Slot             Slot   `json:"slot"`
	PaymentIntentID  string `json:"payment_intent_id"`
	ClientSecret     string `json:"client_secret"`
	Requires3DS      bool   `json:"requires_3ds"`
	ThreeDSChallenge string `json:"three_ds_challenge_url,omitempty"`
}
