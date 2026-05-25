package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"evidencevault/internal/audit"
	"evidencevault/internal/persistence"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

var ErrDuplicateEvent = errors.New("duplicate stripe event")

type Event struct{ ID, Type string }
type Service struct {
	PriceID, BaseURL, WebhookSecret, SecretKey string
	Audit                                      *audit.Service
	Store                                      persistence.Store
}

func (s *Service) CheckoutURL(_ context.Context, tenantID string) (string, error) {
	if s.SecretKey == "" || s.PriceID == "" || s.BaseURL == "" {
		return "", errors.New("billing unavailable: missing STRIPE_SECRET_KEY, STRIPE_PRICE_ID, or BASE_URL")
	}
	vals := url.Values{"mode": {"subscription"}, "success_url": {s.BaseURL + "/app?success=1"}, "cancel_url": {s.BaseURL + "/app?canceled=1"}, "client_reference_id": {tenantID}, "line_items[0][price]": {s.PriceID}, "line_items[0][quantity]": {"1"}}
	return s.postStripe("https://api.stripe.com/v1/checkout/sessions", vals)
}
func (s *Service) PortalURL(_ context.Context, tenantID string) (string, error) {
	if s.SecretKey == "" {
		return "", errors.New("billing unavailable: missing STRIPE_SECRET_KEY")
	}
	vals := url.Values{"customer": {tenantID}, "return_url": {s.BaseURL + "/app"}}
	return s.postStripe("https://api.stripe.com/v1/billing_portal/sessions", vals)
}
func (s *Service) postStripe(u string, vals url.Values) (string, error) {
	req, _ := http.NewRequest(http.MethodPost, u, strings.NewReader(vals.Encode()))
	req.SetBasicAuth(s.SecretKey, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("stripe error: %s", string(b))
	}
	var out struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return "", err
	}
	return out.URL, nil
}
func (s *Service) VerifyWebhook(r *http.Request) (*Event, []byte, error) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, nil, err
	}
	if !verifySignature(payload, r.Header.Get("Stripe-Signature"), s.WebhookSecret) {
		return nil, payload, errors.New("invalid stripe signature")
	}
	var e Event
	if err := json.Unmarshal(payload, &e); err != nil {
		return nil, payload, err
	}
	return &e, payload, nil
}
func verifySignature(payload []byte, header, secret string) bool {
	if secret == "" {
		return false
	}
	parts := strings.Split(header, ",")
	var t, v1 string
	for _, p := range parts {
		kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
		if len(kv) == 2 && kv[0] == "t" {
			t = kv[1]
		} else if len(kv) == 2 && kv[0] == "v1" {
			v1 = kv[1]
		}
	}
	if t == "" || v1 == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(t + "." + string(payload)))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(v1))
}
func (s *Service) RecordAndProcessEvent(ctx context.Context, event Event, _ []byte) error {
	dup := false
	if err := s.Store.Write(func(st *persistence.State) error {
		_, dup = st.StripeEvents[event.ID]
		if !dup {
			st.StripeEvents[event.ID] = struct{}{}
		}
		return nil
	}); err != nil {
		return err
	}
	if dup {
		return ErrDuplicateEvent
	}
	s.Audit.Log(ctx, "", "", "stripe.webhook_processed", "stripe_event", event.ID, "{\"type\":\""+event.Type+"\"}")
	return nil
}
