package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fksava45/reserve/internal/domain"
)

type StripeHTTPClient struct {
	SecretKey string
	BaseURL   string
	Client    *http.Client
}

func NewStripeHTTPClientFromEnv() *StripeHTTPClient {
	return &StripeHTTPClient{
		SecretKey: os.Getenv("STRIPE_SECRET_KEY"),
		BaseURL:   "https://api.stripe.com/v1",
		Client:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *StripeHTTPClient) CreatePaymentIntent(ctx context.Context, amountRSD int64, slotID string) (PaymentIntent, error) {
	values := url.Values{}
	values.Set("amount", strconv.FormatInt(amountRSD, 10))
	values.Set("currency", "rsd")
	values.Set("capture_method", "manual")
	values.Set("metadata[slot_id]", slotID)
	var response stripePaymentIntent
	if err := s.postForm(ctx, "/payment_intents", values, &response); err != nil {
		return PaymentIntent{}, err
	}
	return response.toDomain(), nil
}

func (s *StripeHTTPClient) Authorize(ctx context.Context, paymentIntentID string) (PaymentIntent, error) {
	var response stripePaymentIntent
	if err := s.get(ctx, "/payment_intents/"+paymentIntentID, &response); err != nil {
		return PaymentIntent{}, err
	}
	intent := response.toDomain()
	if response.Status == "requires_capture" {
		intent.Status = domain.PaymentAuthorized
	}
	return intent, nil
}

func (s *StripeHTTPClient) Capture(ctx context.Context, paymentIntentID string) (PaymentIntent, error) {
	var response stripePaymentIntent
	if err := s.postForm(ctx, "/payment_intents/"+paymentIntentID+"/capture", url.Values{}, &response); err != nil {
		return PaymentIntent{}, err
	}
	intent := response.toDomain()
	intent.Status = domain.PaymentCaptured
	return intent, nil
}

func (s *StripeHTTPClient) Void(ctx context.Context, paymentIntentID string) error {
	var response stripePaymentIntent
	return s.postForm(ctx, "/payment_intents/"+paymentIntentID+"/cancel", url.Values{}, &response)
}

func (s *StripeHTTPClient) postForm(ctx context.Context, path string, values url.Values, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.BaseURL+path, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	return s.do(req, target)
}

func (s *StripeHTTPClient) get(ctx context.Context, path string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.BaseURL+path, nil)
	if err != nil {
		return err
	}
	return s.do(req, target)
}

func (s *StripeHTTPClient) do(req *http.Request, target any) error {
	if s.SecretKey == "" {
		return fmt.Errorf("STRIPE_SECRET_KEY is required")
	}
	req.SetBasicAuth(s.SecretKey, "")
	client := s.Client
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return fmt.Errorf("stripe status %d", res.StatusCode)
	}
	return json.NewDecoder(res.Body).Decode(target)
}

type stripePaymentIntent struct {
	ID           string `json:"id"`
	ClientSecret string `json:"client_secret"`
	Amount       int64  `json:"amount"`
	Status       string `json:"status"`
}

func (p stripePaymentIntent) toDomain() PaymentIntent {
	status := domain.PaymentRequiresAction
	if p.Status == "requires_capture" {
		status = domain.PaymentAuthorized
	} else if p.Status == "succeeded" {
		status = domain.PaymentCaptured
	} else if p.Status == "canceled" {
		status = domain.PaymentVoided
	}
	return PaymentIntent{ID: p.ID, ClientSecret: p.ClientSecret, AmountRSD: p.Amount, Status: status}
}
