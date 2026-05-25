package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFromRequest(t *testing.T) {
	sessionSecret := "test-secret"
	validToken := SignSession("cookie-tenant", "cookie-user", sessionSecret)

	tests := []struct {
		name           string
		envVars        map[string]string
		setupRequest   func(*http.Request)
		expectedCtx    Context
		expectedErrStr string
	}{
		{
			name: "development environment defaults",
			envVars: map[string]string{
				"APP_ENV": "development",
			},
			setupRequest: func(r *http.Request) {},
			expectedCtx:  Context{TenantID: "demo-tenant", UserID: "dev-user"},
		},
		{
			name: "development environment with headers",
			envVars: map[string]string{
				"APP_ENV": "development",
			},
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-Tenant-ID", "custom-tenant")
				r.Header.Set("X-User-ID", "custom-user")
			},
			expectedCtx: Context{TenantID: "custom-tenant", UserID: "custom-user"},
		},
		{
			name: "API key valid",
			envVars: map[string]string{
				"API_KEYS": "key1:tenant1,key2:tenant2",
			},
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-API-Key", "key2")
			},
			expectedCtx: Context{TenantID: "tenant2", UserID: "api-key"},
		},
		{
			name: "API key invalid",
			envVars: map[string]string{
				"API_KEYS": "key1:tenant1",
			},
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-API-Key", "invalid-key")
			},
			expectedErrStr: "invalid api key",
		},
		{
			name:           "missing auth (no cookie, no api key, not development)",
			envVars:        map[string]string{},
			setupRequest:   func(r *http.Request) {},
			expectedErrStr: "missing auth",
		},
		{
			name:    "session auth unavailable (no secret)",
			envVars: map[string]string{},
			setupRequest: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "some-value"})
			},
			expectedErrStr: "session auth unavailable",
		},
		{
			name: "invalid session signature",
			envVars: map[string]string{
				"SESSION_SECRET": sessionSecret,
			},
			setupRequest: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: sessionCookieName, Value: SignSession("t1", "u1", "wrong-secret")})
			},
			expectedErrStr: "invalid session signature",
		},
		{
			name: "valid session",
			envVars: map[string]string{
				"SESSION_SECRET": sessionSecret,
			},
			setupRequest: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: sessionCookieName, Value: validToken})
			},
			expectedCtx: Context{TenantID: "cookie-tenant", UserID: "cookie-user"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("APP_ENV", "")
			t.Setenv("API_KEYS", "")
			t.Setenv("SESSION_SECRET", "")
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.setupRequest != nil {
				tt.setupRequest(req)
			}

			ctx, err := FromRequest(req)
			if tt.expectedErrStr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.expectedErrStr)
				}
				if err.Error() != tt.expectedErrStr {
					t.Errorf("expected error %q, got %q", tt.expectedErrStr, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if ctx != tt.expectedCtx {
					t.Errorf("expected context %+v, got %+v", tt.expectedCtx, ctx)
				}
			}
		})
	}
}
