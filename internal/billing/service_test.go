package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http/httptest"
	"strings"
	"testing"

	"evidencevault/internal/audit"
	"evidencevault/internal/persistence"
)

func TestWebhookVerifyAndIdempotency(t *testing.T) {
	payload := `{"id":"evt_1","type":"invoice.paid"}`
	secret := "whsec_test"
	ts := "1710000000"
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "." + payload))
	sig := hex.EncodeToString(mac.Sum(nil))
	req := httptest.NewRequest("POST", "/webhooks/stripe", strings.NewReader(payload))
	req.Header.Set("Stripe-Signature", "t="+ts+",v1="+sig)
	store := persistence.NewMemoryStore()
	s := &Service{WebhookSecret: secret, Audit: audit.NewService(store), Store: store}
	e, b, err := s.VerifyWebhook(req)
	if err != nil || e.ID != "evt_1" || len(b) == 0 {
		t.Fatal("verify failed")
	}
	if err := s.RecordAndProcessEvent(context.Background(), *e, b); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordAndProcessEvent(context.Background(), *e, b); err != ErrDuplicateEvent {
		t.Fatal("expected duplicate")
	}
}
