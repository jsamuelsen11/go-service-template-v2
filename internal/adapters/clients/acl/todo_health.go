package acl

import (
	"context"
	"fmt"
)

// Name returns the identifier used when this component is registered with a
// [ports.HealthRegistry]. The value "todo-api" matches the service name used
// by the underlying [httpclient.Client] for tracing and metrics.
func (c *TodoClient) Name() string {
	return "todo-api"
}

// HealthCheck reports the downstream TODO API's availability based on the
// circuit breaker state -- no network call is made.
//
// State mapping:
//   - "closed"    -- downstream is operating normally; returns nil.
//   - "half-open" -- circuit breaker is probing recovery; returns a
//     descriptive error indicating degraded state.
//   - "open"      -- downstream is unavailable and the breaker is rejecting
//     requests; returns a descriptive error indicating failure.
//
// This reports downstream status, not service readiness. The service itself
// is always ready to handle requests (it returns proper domain errors when
// the downstream is failing). Tying readiness to downstream health would
// prevent the circuit breaker from ever recovering, because Kubernetes would
// stop routing traffic to this service.
func (c *TodoClient) HealthCheck(_ context.Context) error {
	state := c.req.CircuitBreakerState()
	switch state {
	case "closed":
		return nil
	case "half-open":
		return fmt.Errorf("todo-api: degraded (circuit breaker half-open)")
	case "open":
		return fmt.Errorf("todo-api: failing (circuit breaker open)")
	default:
		return fmt.Errorf("todo-api: unknown circuit breaker state %q", state)
	}
}
