package billing

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/stripe/stripe-go/v78"
	portalsession "github.com/stripe/stripe-go/v78/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v78/checkout/session"
	"github.com/stripe/stripe-go/v78/webhook"
)

type Service struct{ PriceID, BaseURL, WebhookSecret string }

func (s Service) CheckoutURL(customerID string) (string, error) {
	params := &stripe.CheckoutSessionParams{Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)), SuccessURL: stripe.String(s.BaseURL + "/app/billing?success=1"), CancelURL: stripe.String(s.BaseURL + "/app/billing?canceled=1"), Customer: stripe.String(customerID), LineItems: []*stripe.CheckoutSessionLineItemParams{{Price: stripe.String(s.PriceID), Quantity: stripe.Int64(1)}}}
	cs, err := checkoutsession.New(params)
	if err != nil {
		return "", err
	}
	return cs.URL, nil
}
func (s Service) PortalURL(customerID string) (string, error) {
	ps, err := portalsession.New(&stripe.BillingPortalSessionParams{Customer: stripe.String(customerID), ReturnURL: stripe.String(s.BaseURL + "/app/billing")})
	if err != nil {
		return "", err
	}
	return ps.URL, nil
}
func (s Service) VerifyWebhook(r *http.Request) (*stripe.Event, []byte, error) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, nil, err
	}
	event, err := webhook.ConstructEvent(payload, r.Header.Get("Stripe-Signature"), s.WebhookSecret)
	if err != nil {
		return nil, payload, err
	}
	return &event, payload, nil
}
func EventObjectRaw(e stripe.Event) string { b, _ := json.Marshal(e.Data.Object); return string(b) }
