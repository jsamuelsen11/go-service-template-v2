// Package health provides a thread-safe health check registry for tracking
// the health of downstream dependencies. The registry is used by the readiness
// endpoint to determine whether the service can accept traffic.
package health

import (
	"context"
	"sync"

	"github.com/jsamuelsen11/go-service-template-v2/internal/ports"
)

// Compile-time interface check.
var _ ports.HealthRegistry = (*Registry)(nil)

// Registry is a thread-safe implementation of [ports.HealthRegistry].
// Components that implement [ports.HealthChecker] are registered at startup
// and checked on each readiness probe.
type Registry struct {
	mu       sync.RWMutex
	checkers []ports.HealthChecker
}

// New creates an empty health check registry.
func New() *Registry {
	return &Registry{}
}

// Register adds a health checker to the registry. Safe for concurrent use.
func (r *Registry) Register(checker ports.HealthChecker) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checkers = append(r.checkers, checker)
}

// CheckAll executes all registered health checks and returns results keyed by
// checker name. Nil values indicate healthy components. The slice is copied
// under a read lock so checks run without holding the lock.
func (r *Registry) CheckAll(ctx context.Context) map[string]error {
	r.mu.RLock()
	checkers := make([]ports.HealthChecker, len(r.checkers))
	copy(checkers, r.checkers)
	r.mu.RUnlock()

	results := make(map[string]error, len(checkers))
	for _, c := range checkers {
		results[c.Name()] = c.HealthCheck(ctx)
	}
	return results
}
