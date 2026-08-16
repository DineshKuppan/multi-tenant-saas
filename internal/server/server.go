// Package server wires up HTTP routes and middleware.
package server

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/redis/go-redis/v9"

	"github.com/DineshKuppan/multi-tenant-saas/internal/db"
	"github.com/DineshKuppan/multi-tenant-saas/internal/middleware"
	"github.com/DineshKuppan/multi-tenant-saas/internal/tenant"
)

func New(database *db.DB, redisClient *redis.Client, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", handleHealthz)
	mux.Handle("GET /v1/ping", middleware.DevOnlyTenantFromHeader(http.HandlerFunc(handlePing)))

	var handler http.Handler = mux
	handler = middleware.Logging(logger)(handler)
	return handler
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handlePing demonstrates the required pattern for tenant-facing handlers:
// the tenant ID comes only from context (set by tenant-extraction
// middleware), never read directly off the request by the handler itself.
func handlePing(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"tenant_id": tenantID, "message": "pong"})
}
