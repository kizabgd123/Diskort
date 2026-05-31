package domain

import "time"

type SlotStatus string
type BookingStatus string
type PaymentStatus string

const (
	SlotAvailable SlotStatus = "AVAILABLE"
	SlotHeld      SlotStatus = "HELD"
	SlotBooked    SlotStatus = "BOOKED"

	BookingPending   BookingStatus = "PENDING"
	BookingConfirmed BookingStatus = "CONFIRMED"
	BookingFailed    BookingStatus = "FAILED"
	BookingCancelled BookingStatus = "CANCELLED"

	PaymentRequiresAction PaymentStatus = "REQUIRES_ACTION"
	PaymentAuthorized     PaymentStatus = "AUTHORIZED"
	PaymentCaptured       PaymentStatus = "CAPTURED"
	PaymentVoided         PaymentStatus = "VOIDED"
)

type Slot struct {
	ID       string     `json:"slot_id"`
	CourtID  string     `json:"court_id"`
	StartAt  time.Time  `json:"start_at"`
	EndAt    time.Time  `json:"end_at"`
	PriceRSD int64      `json:"price_rsd"`
	Status   SlotStatus `json:"status"`
}

type Customer struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

type Booking struct {
	ID                string        `json:"booking_id"`
	GoogleBookingID   string        `json:"google_booking_id,omitempty"`
	SlotID            string        `json:"slot_id"`
	Customer          Customer      `json:"customer"`
	Status            BookingStatus `json:"status"`
	PaymentIntentID   string        `json:"payment_intent_id"`
	PaymentStatus     PaymentStatus `json:"payment_status"`
	ConfirmationToken string        `json:"confirmation_token,omitempty"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

type CheckoutSession struct {
	SessionID       string `json:"checkout_session_id"`
	PaymentIntentID string `json:"payment_intent_id"`
	ClientSecret    string `json:"client_secret"`
	Requires3DS     bool   `json:"requires_3ds"`
	SCAURL          string `json:"sca_url,omitempty"`
	AmountRSD       int64  `json:"amount_rsd"`
}
