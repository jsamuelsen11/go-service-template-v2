package ports

import "context"

// HealthChecker is implemented by any component that can report its health.
// Examples: downstream API clients, database connections, cache connections.
type HealthChecker interface {
	// Name returns a human-readable identifier for this component
	// (e.g., "todo-api", "database", "redis").
	Name() string

	// HealthCheck performs the health check and returns nil if healthy,
	// or an error describing the failure.
	// Implementations should respect context cancellation and deadlines.
	HealthCheck(ctx context.Context) error
}

// HealthRegistry manages registration and execution of health checkers.
// Used by the readiness endpoint handler to determine service readiness.
type HealthRegistry interface {
	// Register adds a HealthChecker to the registry.
	Register(checker HealthChecker)

	// CheckAll executes all registered health checks and returns results
	// keyed by checker name. Nil values indicate healthy components.
	CheckAll(ctx context.Context) map[string]error
}
