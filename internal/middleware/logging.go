package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

type logFieldsKey struct{}

type logFields struct {
	tenantID string
}

// SetTenantID records the tenant ID for the current request's log line.
// Tenant-extraction middleware nested inside Logging must call this
// (in addition to tenant.WithID, which handlers read from) because
// net/http's context changes don't propagate back up through
// next.ServeHTTP to an outer middleware — Logging can't just re-read
// r.Context() after next returns and expect to see values set below it.
func SetTenantID(ctx context.Context, id string) {
	if f, ok := ctx.Value(logFieldsKey{}).(*logFields); ok {
		f.tenantID = id
	}
}

// Logging logs one structured line per request, including tenant_id when
// present, so production incidents can be filtered by tenant from the start.
func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			fields := &logFields{}
			ctx := context.WithValue(r.Context(), logFieldsKey{}, fields)

			next.ServeHTTP(rec, r.WithContext(ctx))

			logger.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration_ms", time.Since(start).Milliseconds(),
				"tenant_id", fields.tenantID,
			)
		})
	}
}
