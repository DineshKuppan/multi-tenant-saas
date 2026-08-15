package middleware

import (
	"net/http"

	"multitenantsaas/internal/tenant"
)

// DevOnlyTenantFromHeader reads the tenant ID from a plain header so local
// endpoints are exercisable before auth exists.
//
// THIS IS NOT SAFE FOR PRODUCTION: it trusts a client-supplied header. It
// must be replaced with middleware that derives the tenant ID from a
// validated JWT claim before any tenant-facing route is exposed outside
// local development. Do not extend this middleware's use past scaffolding.
func DevOnlyTenantFromHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Header.Get("X-Debug-Tenant-ID")
		if tenantID == "" {
			http.Error(w, "tenant context missing", http.StatusUnauthorized)
			return
		}
		ctx := tenant.WithID(r.Context(), tenantID)
		SetTenantID(ctx, tenantID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
