package auth

import "net/http"

func TenantIDFromRequest(r *http.Request) string {
	t := r.Header.Get("X-Tenant-ID")
	if t == "" {
		return "demo-tenant"
	}
	return t
}
