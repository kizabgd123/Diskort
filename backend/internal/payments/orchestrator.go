package payments

import (
	"context"
	"errors"
	"fmt"
)

type Intent struct {
	ID           string `json:"id"`
	ClientSecret string `json:"client_secret"`
	AmountMinor  int64  `json:"amount_minor"`
	Currency     string `json:"currency"`
	Requires3DS  bool   `json:"requires_3ds"`
}

type Authorization struct {
	ID         string
	IntentID   string
	Authorized bool
}

type Capture struct {
	ID       string
	IntentID string
	Captured bool
}

type PSP interface {
	CreateIntent(ctx context.Context, amountMinor int64, currency, idempotencyKey string) (Intent, error)
	Authorize(ctx context.Context, intentID string) (Authorization, error)
	Capture(ctx context.Context, authorizationID string) (Capture, error)
	Cancel(ctx context.Context, intentID string) error
}

type SCAService interface {
	ChallengeURL(ctx context.Context, intent Intent) (string, error)
}

type Orchestrator struct {
	psp PSP
	sca SCAService
}

func NewOrchestrator(psp PSP, sca SCAService) *Orchestrator { return &Orchestrator{psp: psp, sca: sca} }

func (o *Orchestrator) PrepareCheckout(ctx context.Context, amountMinor int64, currency, idempotencyKey string) (Intent, string, error) {
	intent, err := o.psp.CreateIntent(ctx, amountMinor, currency, idempotencyKey)
	if err != nil {
		return Intent{}, "", err
	}
	if !intent.Requires3DS {
		return intent, "", nil
	}
	challengeURL, err := o.sca.ChallengeURL(ctx, intent)
	return intent, challengeURL, err
}

func (o *Orchestrator) Authorize(ctx context.Context, intentID string) (Authorization, error) {
	auth, err := o.psp.Authorize(ctx, intentID)
	if err != nil {
		return Authorization{}, err
	}
	if !auth.Authorized {
		return Authorization{}, errors.New("payment authorization declined")
	}
	return auth, nil
}

func (o *Orchestrator) Capture(ctx context.Context, authorizationID string) (Capture, error) {
	capture, err := o.psp.Capture(ctx, authorizationID)
	if err != nil {
		return Capture{}, err
	}
	if !capture.Captured {
		return Capture{}, errors.New("payment capture failed")
	}
	return capture, nil
}

func (o *Orchestrator) Rollback(ctx context.Context, intentID string) error {
	return o.psp.Cancel(ctx, intentID)
}

type FakeStripe struct{}

func (FakeStripe) CreateIntent(ctx context.Context, amountMinor int64, currency, idempotencyKey string) (Intent, error) {
	if amountMinor <= 0 {
		return Intent{}, errors.New("amount must be positive")
	}
	return Intent{ID: "pi_" + idempotencyKey, ClientSecret: "secret_" + idempotencyKey, AmountMinor: amountMinor, Currency: currency, Requires3DS: amountMinor >= 500000}, nil
}
func (FakeStripe) Authorize(ctx context.Context, intentID string) (Authorization, error) {
	return Authorization{ID: "auth_" + intentID, IntentID: intentID, Authorized: true}, nil
}
func (FakeStripe) Capture(ctx context.Context, authorizationID string) (Capture, error) {
	return Capture{ID: "cap_" + authorizationID, IntentID: authorizationID, Captured: true}, nil
}
func (FakeStripe) Cancel(ctx context.Context, intentID string) error { return nil }

type ThreeDS struct{ BaseURL string }

func (s ThreeDS) ChallengeURL(ctx context.Context, intent Intent) (string, error) {
	base := s.BaseURL
	if base == "" {
		base = "https://sca.example.test/challenge"
	}
	return fmt.Sprintf("%s/%s", base, intent.ID), nil
}
