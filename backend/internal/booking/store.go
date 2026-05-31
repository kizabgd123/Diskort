package booking

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

var (
	ErrSlotNotFound    = errors.New("slot not found")
	ErrSlotUnavailable = errors.New("slot unavailable")
	ErrBookingNotFound = errors.New("booking not found")
	ErrPaymentNotReady = errors.New("payment is not authorized")
)

type Store interface {
	ListAvailability(ctx context.Context, from, to time.Time) ([]Slot, error)
	GetSlot(ctx context.Context, slotID string) (Slot, error)
	MarkSlotBooked(ctx context.Context, slotID string) error
	CreateBooking(ctx context.Context, booking Booking) (Booking, error)
	GetBooking(ctx context.Context, bookingID string) (Booking, error)
	FailBooking(ctx context.Context, bookingID string) error
}

type InMemoryStore struct {
	mu       sync.RWMutex
	slots    map[string]Slot
	bookings map[string]Booking
}

func NewInMemoryStore(slots []Slot) *InMemoryStore {
	indexed := make(map[string]Slot, len(slots))
	for _, slot := range slots {
		indexed[slot.ID] = slot
	}
	return &InMemoryStore{slots: indexed, bookings: map[string]Booking{}}
}

func DefaultSlots(now time.Time) []Slot {
	base := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, time.FixedZone("Europe/Belgrade", 3600))
	courts := []struct {
		id, name, sport string
		price           int64
	}{
		{"pitch-5a", "Mini pitch A", "Football 5v5", 480000},
		{"pitch-5b", "Mini pitch B", "Football 5v5", 480000},
		{"pitch-7", "Main pitch", "Football 7v7", 720000},
	}
	var slots []Slot
	for d := 0; d < 14; d++ {
		for _, court := range courts {
			for hour := 8; hour < 23; hour++ {
				start := base.AddDate(0, 0, d).Add(time.Duration(hour-8) * time.Hour)
				slots = append(slots, Slot{ID: court.id + "_" + start.Format("20060102T1504"), CourtID: court.id, CourtName: court.name, Sport: court.sport, StartsAt: start, EndsAt: start.Add(time.Hour), PriceMinor: court.price, Currency: "RSD", Status: SlotAvailable})
			}
		}
	}
	return slots
}

func (s *InMemoryStore) ListAvailability(ctx context.Context, from, to time.Time) ([]Slot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Slot
	for _, slot := range s.slots {
		if (slot.StartsAt.Equal(from) || slot.StartsAt.After(from)) && slot.StartsAt.Before(to) && slot.Status == SlotAvailable {
			out = append(out, slot)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartsAt.Before(out[j].StartsAt) })
	return out, nil
}

func (s *InMemoryStore) GetSlot(ctx context.Context, slotID string) (Slot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	slot, ok := s.slots[slotID]
	if !ok {
		return Slot{}, ErrSlotNotFound
	}
	if slot.Status != SlotAvailable {
		return Slot{}, ErrSlotUnavailable
	}
	return slot, nil
}

func (s *InMemoryStore) MarkSlotBooked(ctx context.Context, slotID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	slot, ok := s.slots[slotID]
	if !ok {
		return ErrSlotNotFound
	}
	if slot.Status != SlotAvailable {
		return ErrSlotUnavailable
	}
	slot.Status = SlotBooked
	s.slots[slotID] = slot
	return nil
}

func (s *InMemoryStore) CreateBooking(ctx context.Context, booking Booking) (Booking, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.bookings[booking.ID]; exists {
		return Booking{}, ErrSlotUnavailable
	}
	s.bookings[booking.ID] = booking
	return booking, nil
}

func (s *InMemoryStore) GetBooking(ctx context.Context, bookingID string) (Booking, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	booking, ok := s.bookings[bookingID]
	if !ok {
		return Booking{}, ErrBookingNotFound
	}
	return booking, nil
}

func (s *InMemoryStore) FailBooking(ctx context.Context, bookingID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	booking, ok := s.bookings[bookingID]
	if !ok {
		return ErrBookingNotFound
	}
	booking.Status = BookingFailed
	booking.UpdatedAt = time.Now().UTC()
	s.bookings[bookingID] = booking
	return nil
}
