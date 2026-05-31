package booking

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/fksava45/reserve/internal/domain"
)

var (
	ErrSlotNotFound     = errors.New("slot not found")
	ErrSlotUnavailable  = errors.New("slot unavailable")
	ErrBookingNotFound  = errors.New("booking not found")
	ErrDuplicateBooking = errors.New("duplicate booking")
	ErrLockNotAcquired  = errors.New("slot lock not acquired")
	ErrPaymentRequired  = errors.New("payment authorization required")
	ErrPaymentFailed    = errors.New("payment failed")
)

type Repository interface {
	ListAvailability(ctx context.Context, from, to time.Time) ([]domain.Slot, error)
	GetSlot(ctx context.Context, slotID string) (domain.Slot, error)
	CreateBooking(ctx context.Context, booking domain.Booking) (domain.Booking, error)
	GetBooking(ctx context.Context, bookingID string) (domain.Booking, error)
	UpdateBooking(ctx context.Context, booking domain.Booking) error
	MarkSlotBooked(ctx context.Context, slotID string) error
	MarkSlotAvailable(ctx context.Context, slotID string) error
}

type LockManager interface {
	Acquire(ctx context.Context, key, owner string, ttl time.Duration) (bool, error)
	Release(ctx context.Context, key, owner string) error
}

type MemoryRepository struct {
	mu       sync.RWMutex
	slots    map[string]domain.Slot
	bookings map[string]domain.Booking
}

func NewMemoryRepository(slots []domain.Slot) *MemoryRepository {
	items := make(map[string]domain.Slot, len(slots))
	for _, slot := range slots {
		items[slot.ID] = slot
	}
	return &MemoryRepository{slots: items, bookings: map[string]domain.Booking{}}
}

func SeedSlots(now time.Time) []domain.Slot {
	base := time.Date(now.Year(), now.Month(), now.Day()+1, 8, 0, 0, 0, time.UTC)
	courts := []struct {
		id    string
		price int64
	}{
		{"pitch-5a", 4800},
		{"pitch-5b", 4800},
		{"pitch-7", 7200},
	}
	var slots []domain.Slot
	for _, court := range courts {
		for h := 0; h < 15; h++ {
			start := base.Add(time.Duration(h) * time.Hour)
			slots = append(slots, domain.Slot{
				ID:       court.id + "_" + start.Format("20060102T1504"),
				CourtID:  court.id,
				StartAt:  start,
				EndAt:    start.Add(time.Hour),
				PriceRSD: court.price,
				Status:   domain.SlotAvailable,
			})
		}
	}
	return slots
}

func (r *MemoryRepository) ListAvailability(ctx context.Context, from, to time.Time) ([]domain.Slot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []domain.Slot
	for _, slot := range r.slots {
		if !slot.StartAt.Before(from) && slot.StartAt.Before(to) {
			out = append(out, slot)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartAt.Before(out[j].StartAt) })
	return out, nil
}

func (r *MemoryRepository) GetSlot(ctx context.Context, slotID string) (domain.Slot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	slot, ok := r.slots[slotID]
	if !ok {
		return domain.Slot{}, ErrSlotNotFound
	}
	return slot, nil
}

func (r *MemoryRepository) CreateBooking(ctx context.Context, item domain.Booking) (domain.Booking, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if slot, ok := r.slots[item.SlotID]; !ok || slot.Status != domain.SlotAvailable {
		return domain.Booking{}, ErrSlotUnavailable
	}
	for _, existing := range r.bookings {
		if existing.SlotID == item.SlotID && existing.Status == domain.BookingConfirmed {
			return domain.Booking{}, ErrDuplicateBooking
		}
	}
	r.bookings[item.ID] = item
	return item, nil
}

func (r *MemoryRepository) GetBooking(ctx context.Context, bookingID string) (domain.Booking, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	booking, ok := r.bookings[bookingID]
	if !ok {
		return domain.Booking{}, ErrBookingNotFound
	}
	return booking, nil
}

func (r *MemoryRepository) UpdateBooking(ctx context.Context, item domain.Booking) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.bookings[item.ID]; !ok {
		return ErrBookingNotFound
	}
	item.UpdatedAt = time.Now().UTC()
	r.bookings[item.ID] = item
	return nil
}

func (r *MemoryRepository) MarkSlotBooked(ctx context.Context, slotID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	slot, ok := r.slots[slotID]
	if !ok {
		return ErrSlotNotFound
	}
	if slot.Status != domain.SlotAvailable {
		return ErrSlotUnavailable
	}
	slot.Status = domain.SlotBooked
	r.slots[slotID] = slot
	return nil
}

func (r *MemoryRepository) MarkSlotAvailable(ctx context.Context, slotID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	slot, ok := r.slots[slotID]
	if !ok {
		return ErrSlotNotFound
	}
	slot.Status = domain.SlotAvailable
	r.slots[slotID] = slot
	return nil
}
