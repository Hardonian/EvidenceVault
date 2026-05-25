package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestFromRequest(t *testing.T) {
	tests := []struct {
		name          string
		setupRequest  func(*http.Request)
		setupEnv      func(t *testing.T)
		expectedCtx   Context
		expectedError string
	}{
		// Development mode tests
		{
			name: "development mode - no headers",
			setupRequest: func(r *http.Request) {},
			setupEnv: func(t *testing.T) {
				t.Setenv("APP_ENV", "development")
			},
			expectedCtx: Context{TenantID: "demo-tenant", UserID: "dev-user"},
		},
		{
			name: "development mode - custom headers",
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-Tenant-ID", "custom-tenant")
				r.Header.Set("X-User-ID", "custom-user")
			},
			setupEnv: func(t *testing.T) {
				t.Setenv("APP_ENV", "development")
			},
			expectedCtx: Context{TenantID: "custom-tenant", UserID: "custom-user"},
		},

		// API Key tests
		{
			name: "production mode - valid api key",
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-API-Key", "my-secret-key")
			},
			setupEnv: func(t *testing.T) {
				t.Setenv("API_KEYS", "my-secret-key:tenant-from-api")
			},
			expectedCtx: Context{TenantID: "tenant-from-api", UserID: "api-key"},
		},
		{
			name: "production mode - invalid api key",
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-API-Key", "invalid-key")
			},
			setupEnv: func(t *testing.T) {
				t.Setenv("API_KEYS", "my-secret-key:tenant-from-api")
			},
			expectedError: "invalid api key",
		},

		// Session Cookie tests
		{
			name: "production mode - missing auth",
			setupRequest: func(r *http.Request) {},
			setupEnv: func(t *testing.T) {},
			expectedError: "missing auth",
		},
		{
			name: "production mode - valid session cookie",
			setupRequest: func(r *http.Request) {
				cookieValue := SignSession("session-tenant", "session-user", "my-secret")
				r.AddCookie(&http.Cookie{Name: "ev_session", Value: cookieValue})
			},
			setupEnv: func(t *testing.T) {
				t.Setenv("SESSION_SECRET", "my-secret")
			},
			expectedCtx: Context{TenantID: "session-tenant", UserID: "session-user"},
		},
		{
			name: "production mode - session cookie missing secret",
			setupRequest: func(r *http.Request) {
				cookieValue := SignSession("session-tenant", "session-user", "my-secret")
				r.AddCookie(&http.Cookie{Name: "ev_session", Value: cookieValue})
			},
			setupEnv: func(t *testing.T) {
				// No SESSION_SECRET set
			},
			expectedError: "session auth unavailable",
		},
		{
			name: "production mode - invalid session signature",
			setupRequest: func(r *http.Request) {
				cookieValue := SignSession("session-tenant", "session-user", "wrong-secret")
				r.AddCookie(&http.Cookie{Name: "ev_session", Value: cookieValue})
			},
			setupEnv: func(t *testing.T) {
				t.Setenv("SESSION_SECRET", "my-secret") // Different secret
			},
			expectedError: "invalid session signature",
		},
		{
			name: "production mode - invalid session format",
			setupRequest: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: "ev_session", Value: "invalid-base64"})
			},
			setupEnv: func(t *testing.T) {
				t.Setenv("SESSION_SECRET", "my-secret")
			},
			expectedError: "invalid session",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Clear all env vars that could affect tests before setting test-specific ones
			os.Unsetenv("APP_ENV")
			os.Unsetenv("API_KEYS")
			os.Unsetenv("SESSION_SECRET")

			tc.setupEnv(t)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			tc.setupRequest(req)

			ctx, err := FromRequest(req)

			if tc.expectedError != "" {
				if err == nil {
					t.Fatalf("expected error '%s', got nil", tc.expectedError)
				}
				if err.Error() != tc.expectedError {
					t.Fatalf("expected error '%s', got '%s'", tc.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got '%v'", err)
				}
				if ctx.TenantID != tc.expectedCtx.TenantID {
					t.Errorf("expected TenantID '%s', got '%s'", tc.expectedCtx.TenantID, ctx.TenantID)
				}
				if ctx.UserID != tc.expectedCtx.UserID {
					t.Errorf("expected UserID '%s', got '%s'", tc.expectedCtx.UserID, ctx.UserID)
				}
			}
		})
	}
}
