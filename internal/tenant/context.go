// Package tenant carries the authenticated request's tenant ID through
// context.Context so every downstream call (query, cache key, log line) can
// be scoped to it explicitly.
package tenant

import (
	"context"
	"errors"
)

type ctxKey struct{}

var ErrNoTenant = errors.New("tenant: no tenant in context")

func WithID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// FromContext returns the tenant ID stored by WithID, or ErrNoTenant if none
// was set. Callers on tenant-facing paths must treat ErrNoTenant as an
// authorization failure, not a value to default around.
func FromContext(ctx context.Context) (string, error) {
	id, ok := ctx.Value(ctxKey{}).(string)
	if !ok || id == "" {
		return "", ErrNoTenant
	}
	return id, nil
}
