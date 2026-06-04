package payments

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type StripeClient struct {
	SecretKey  string
	HTTPClient *http.Client
}

func (s StripeClient) CreateIntent(ctx context.Context, amountMinor int64, currency, idempotencyKey string) (Intent, error) {
	values := url.Values{}
	values.Set("amount", strconv.FormatInt(amountMinor, 10))
	values.Set("currency", strings.ToLower(currency))
	values.Set("capture_method", "manual")
	values.Set("automatic_payment_methods[enabled]", "true")
	var out struct {
		ID           string `json:"id"`
		ClientSecret string `json:"client_secret"`
		NextAction   any    `json:"next_action"`
	}
	if err := s.post(ctx, "/v1/payment_intents", values, idempotencyKey, &out); err != nil {
		return Intent{}, err
	}
	return Intent{ID: out.ID, ClientSecret: out.ClientSecret, AmountMinor: amountMinor, Currency: currency, Requires3DS: out.NextAction != nil}, nil
}

func (s StripeClient) Authorize(ctx context.Context, intentID string) (Authorization, error) {
	var out struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := s.post(ctx, "/v1/payment_intents/"+url.PathEscape(intentID)+"/confirm", url.Values{}, "authorize_"+intentID, &out); err != nil {
		return Authorization{}, err
	}
	return Authorization{ID: "auth_" + out.ID, IntentID: out.ID, Authorized: out.Status == "requires_capture" || out.Status == "succeeded"}, nil
}

func (s StripeClient) Capture(ctx context.Context, authorizationID string) (Capture, error) {
	intentID := strings.TrimPrefix(authorizationID, "auth_")
	var out struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := s.post(ctx, "/v1/payment_intents/"+url.PathEscape(intentID)+"/capture", url.Values{}, "capture_"+intentID, &out); err != nil {
		return Capture{}, err
	}
	return Capture{ID: "cap_" + out.ID, IntentID: out.ID, Captured: out.Status == "succeeded"}, nil
}

func (s StripeClient) Cancel(ctx context.Context, intentID string) error {
	var out map[string]any
	return s.post(ctx, "/v1/payment_intents/"+url.PathEscape(intentID)+"/cancel", url.Values{}, "cancel_"+intentID, &out)
}

func (s StripeClient) post(ctx context.Context, path string, values url.Values, idempotencyKey string, out any) error {
	if s.SecretKey == "" {
		return errors.New("stripe secret key is required")
	}
	client := s.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.stripe.com"+path, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.SetBasicAuth(s.SecretKey, "")
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return errors.New("stripe request failed with status " + resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
