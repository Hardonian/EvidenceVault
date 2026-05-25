package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
)

const (
	sessionCookieName = "ev_session"
)

type Context struct {
	TenantID string
	UserID   string
}

var (
	apiKeysMap  map[string]string
	apiKeysOnce sync.Once
)

func getAPIKeys() map[string]string {
	apiKeysOnce.Do(func() {
		apiKeysMap = parseAPIKeyMapping(os.Getenv("API_KEYS"))
	})
	return apiKeysMap
}

func FromRequest(r *http.Request) (Context, error) {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "development" {
		tenant := r.Header.Get("X-Tenant-ID")
		if tenant == "" {
			tenant = "demo-tenant"
		}
		user := r.Header.Get("X-User-ID")
		if user == "" {
			user = "dev-user"
		}
		return Context{TenantID: tenant, UserID: user}, nil
	}

	if api := r.Header.Get("X-API-Key"); api != "" {
		tenant, ok := getAPIKeys()[api]
		if !ok {
			return Context{}, errors.New("invalid api key")
		}
		return Context{TenantID: tenant, UserID: "api-key"}, nil
	}

	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return Context{}, errors.New("missing auth")
	}
	secret := os.Getenv("SESSION_SECRET")
	if secret == "" {
		return Context{}, errors.New("session auth unavailable")
	}
	ctx, err := verifySession(cookie.Value, secret)
	if err != nil {
		return Context{}, err
	}
	return ctx, nil
}

func parseAPIKeyMapping(v string) map[string]string {
	m := map[string]string{}
	for _, pair := range strings.Split(v, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			continue
		}
		m[parts[0]] = parts[1]
	}
	return m
}

func verifySession(token, secret string) (Context, error) {
	raw, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return Context{}, errors.New("invalid session")
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 3 {
		return Context{}, errors.New("invalid session")
	}
	payload := parts[0] + "|" + parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(parts[2]), []byte(expected)) {
		return Context{}, errors.New("invalid session signature")
	}
	if parts[0] == "" || parts[1] == "" {
		return Context{}, errors.New("invalid session claims")
	}
	return Context{TenantID: parts[0], UserID: parts[1]}, nil
}

func SignSession(tenantID, userID, secret string) string {
	payload := fmt.Sprintf("%s|%s", tenantID, userID)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return base64.StdEncoding.EncodeToString([]byte(payload + "|" + sig))
}
