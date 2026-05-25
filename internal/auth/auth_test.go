package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFromRequest(t *testing.T) {
	validSecret := "super-secret-key-that-is-long-enough"
	validCookieToken := SignSession("tenant-123", "user-456", validSecret)

	tests := []struct {
		name          string
		env           map[string]string
		setupRequest  func(*http.Request)
		wantCtx       Context
		wantErrString string
	}{
		{
			name: "dev env - default headers",
			env: map[string]string{
				"APP_ENV": "development",
			},
			setupRequest: func(r *http.Request) {},
			wantCtx: Context{
				TenantID: "demo-tenant",
				UserID:   "dev-user",
			},
		},
		{
			name: "dev env - custom headers",
			env: map[string]string{
				"APP_ENV": "development",
			},
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-Tenant-ID", "custom-tenant")
				r.Header.Set("X-User-ID", "custom-user")
			},
			wantCtx: Context{
				TenantID: "custom-tenant",
				UserID:   "custom-user",
			},
		},
		{
			name: "prod env - valid api key",
			env: map[string]string{
				"API_KEYS": "key1:tenant1,key2:tenant2",
			},
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-API-Key", "key2")
			},
			wantCtx: Context{
				TenantID: "tenant2",
				UserID:   "api-key",
			},
		},
		{
			name: "prod env - invalid api key",
			env: map[string]string{
				"API_KEYS": "key1:tenant1",
			},
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-API-Key", "key2")
			},
			wantErrString: "invalid api key",
		},
		{
			name: "prod env - missing auth completely",
			env:  map[string]string{},
			setupRequest: func(r *http.Request) {
			},
			wantErrString: "missing auth",
		},
		{
			name: "prod env - valid session cookie",
			env: map[string]string{
				"SESSION_SECRET": validSecret,
			},
			setupRequest: func(r *http.Request) {
				r.AddCookie(&http.Cookie{
					Name:  sessionCookieName,
					Value: validCookieToken,
				})
			},
			wantCtx: Context{
				TenantID: "tenant-123",
				UserID:   "user-456",
			},
		},
		{
			name: "prod env - valid session cookie, missing secret",
			env: map[string]string{
				"SESSION_SECRET": "",
			},
			setupRequest: func(r *http.Request) {
				r.AddCookie(&http.Cookie{
					Name:  sessionCookieName,
					Value: validCookieToken,
				})
			},
			wantErrString: "session auth unavailable",
		},
		{
			name: "prod env - invalid session cookie",
			env: map[string]string{
				"SESSION_SECRET": validSecret,
			},
			setupRequest: func(r *http.Request) {
				r.AddCookie(&http.Cookie{
					Name:  sessionCookieName,
					Value: "invalid-token",
				})
			},
			wantErrString: "invalid session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment variables that might affect the test
			t.Setenv("APP_ENV", "")
			t.Setenv("API_KEYS", "")
			t.Setenv("SESSION_SECRET", "")

			// Set the environment variables for this test case
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.setupRequest != nil {
				tt.setupRequest(req)
			}

			ctx, err := FromRequest(req)

			if tt.wantErrString != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErrString)
				}
				if err.Error() != tt.wantErrString {
					t.Errorf("expected error %q, got %q", tt.wantErrString, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if ctx != tt.wantCtx {
				t.Errorf("expected context %+v, got %+v", tt.wantCtx, ctx)
			}
		})
	}
}
