package payment

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fksava45/reserve/internal/domain"
)

type StripeClient interface {
	CreatePaymentIntent(ctx context.Context, amountRSD int64, slotID string) (PaymentIntent, error)
	Authorize(ctx context.Context, paymentIntentID string) (PaymentIntent, error)
	Capture(ctx context.Context, paymentIntentID string) (PaymentIntent, error)
	Void(ctx context.Context, paymentIntentID string) error
}

type SCAClient interface {
	Challenge(ctx context.Context, paymentIntentID string, customer domain.Customer) (ChallengeResult, error)
}

type PaymentIntent struct {
	ID           string
	ClientSecret string
	AmountRSD    int64
	Status       domain.PaymentStatus
}

type ChallengeResult struct {
	Required      bool
	Authenticated bool
	RedirectURL   string
}

type Orchestrator struct {
	stripe StripeClient
	sca    SCAClient
}

func NewOrchestrator(stripe StripeClient, sca SCAClient) *Orchestrator {
	return &Orchestrator{stripe: stripe, sca: sca}
}

func (o *Orchestrator) Checkout(ctx context.Context, slot domain.Slot, customer domain.Customer) (domain.CheckoutSession, error) {
	intent, err := o.stripe.CreatePaymentIntent(ctx, slot.PriceRSD, slot.ID)
	if err != nil {
		return domain.CheckoutSession{}, err
	}
	challenge, err := o.sca.Challenge(ctx, intent.ID, customer)
	if err != nil {
		return domain.CheckoutSession{}, err
	}
	return domain.CheckoutSession{
		SessionID:       "cs_" + intent.ID,
		PaymentIntentID: intent.ID,
		ClientSecret:    intent.ClientSecret,
		Requires3DS:     challenge.Required && !challenge.Authenticated,
		SCAURL:          challenge.RedirectURL,
		AmountRSD:       slot.PriceRSD,
	}, nil
}

func (o *Orchestrator) Authorize(ctx context.Context, paymentIntentID string) (PaymentIntent, error) {
	return o.stripe.Authorize(ctx, paymentIntentID)
}

func (o *Orchestrator) Capture(ctx context.Context, paymentIntentID string) (PaymentIntent, error) {
	return o.stripe.Capture(ctx, paymentIntentID)
}

func (o *Orchestrator) Void(ctx context.Context, paymentIntentID string) error {
	return o.stripe.Void(ctx, paymentIntentID)
}

type FakeStripe struct {
	mu      sync.Mutex
	counter int
	items   map[string]PaymentIntent
}

func NewFakeStripe() *FakeStripe {
	return &FakeStripe{items: map[string]PaymentIntent{}}
}

func (s *FakeStripe) CreatePaymentIntent(ctx context.Context, amountRSD int64, slotID string) (PaymentIntent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter++
	intent := PaymentIntent{
		ID:           fmt.Sprintf("pi_%06d", s.counter),
		ClientSecret: fmt.Sprintf("pi_%06d_secret", s.counter),
		AmountRSD:    amountRSD,
		Status:       domain.PaymentRequiresAction,
	}
	s.items[intent.ID] = intent
	return intent, nil
}

func (s *FakeStripe) Authorize(ctx context.Context, paymentIntentID string) (PaymentIntent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	intent, ok := s.items[paymentIntentID]
	if !ok {
		return PaymentIntent{}, fmt.Errorf("payment intent %s not found", paymentIntentID)
	}
	intent.Status = domain.PaymentAuthorized
	s.items[paymentIntentID] = intent
	return intent, nil
}

func (s *FakeStripe) Capture(ctx context.Context, paymentIntentID string) (PaymentIntent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	intent, ok := s.items[paymentIntentID]
	if !ok {
		return PaymentIntent{}, fmt.Errorf("payment intent %s not found", paymentIntentID)
	}
	if intent.Status != domain.PaymentAuthorized {
		return PaymentIntent{}, fmt.Errorf("payment intent %s is not authorized", paymentIntentID)
	}
	intent.Status = domain.PaymentCaptured
	s.items[paymentIntentID] = intent
	return intent, nil
}

func (s *FakeStripe) Void(ctx context.Context, paymentIntentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	intent, ok := s.items[paymentIntentID]
	if !ok {
		return nil
	}
	intent.Status = domain.PaymentVoided
	s.items[paymentIntentID] = intent
	return nil
}

type FakeSCA struct{}

func (FakeSCA) Challenge(ctx context.Context, paymentIntentID string, customer domain.Customer) (ChallengeResult, error) {
	return ChallengeResult{
		Required:      true,
		Authenticated: true,
		RedirectURL:   "https://sca.example.test/3ds/" + paymentIntentID + "?issued_at=" + time.Now().UTC().Format(time.RFC3339),
	}, nil
}
