package billing

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stripe/stripe-go/v78"
	portalsession "github.com/stripe/stripe-go/v78/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v78/checkout/session"
	"github.com/stripe/stripe-go/v78/webhook"
)

var ErrDuplicateEvent = errors.New("duplicate stripe event")

type Service struct {
	PriceID, BaseURL, WebhookSecret string
	DB                              *pgxpool.Pool
}

func (s *Service) CheckoutURL(customerID string) (string, error) {
	params := &stripe.CheckoutSessionParams{Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)), SuccessURL: stripe.String(s.BaseURL + "/app?success=1"), CancelURL: stripe.String(s.BaseURL + "/app?canceled=1"), Customer: stripe.String(customerID), LineItems: []*stripe.CheckoutSessionLineItemParams{{Price: stripe.String(s.PriceID), Quantity: stripe.Int64(1)}}}
	cs, err := checkoutsession.New(params)
	if err != nil {
		return "", err
	}
	return cs.URL, nil
}
func (s *Service) PortalURL(customerID string) (string, error) {
	ps, err := portalsession.New(&stripe.BillingPortalSessionParams{Customer: stripe.String(customerID), ReturnURL: stripe.String(s.BaseURL + "/app")})
	if err != nil {
		return "", err
	}
	return ps.URL, nil
}
func (s *Service) VerifyWebhook(r *http.Request) (*stripe.Event, []byte, error) {
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

func (s *Service) RecordAndProcessEvent(ctx context.Context, event stripe.Event, payload []byte) error {
	_, err := s.DB.Exec(ctx, `insert into stripe_events (stripe_event_id, event_type, payload, status, created_at) values ($1,$2,$3,'received',now())`, event.ID, event.Type, payload)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDuplicateEvent
		}
		return err
	}
	_, err = s.DB.Exec(ctx, `update stripe_events set status='processed', processed_at=$2 where stripe_event_id=$1`, event.ID, time.Now().UTC())
	if err != nil {
		_, _ = s.DB.Exec(ctx, `update stripe_events set status='failed' where stripe_event_id=$1`, event.ID)
		return err
	}
	return nil
}
func EventObjectRaw(e stripe.Event) string { b, _ := json.Marshal(e.Data.Object); return string(b) }
