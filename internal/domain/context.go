package domain

import "context"

// Action represents a single executable operation with rollback capability.
// Implementations should be idempotent where possible to support safe retries.
//
// Action is defined in the domain layer so that domain services can reference
// it without depending on the application layer (dependency inversion).
type Action interface {
	// Execute performs the action. The context carries cancellation and
	// deadline signals that the implementation should respect.
	Execute(ctx context.Context) error

	// Rollback reverses the effect of a previously successful Execute call.
	// Rollback is only called if Execute returned nil. The context may
	// differ from the one passed to Execute.
	Rollback(ctx context.Context) error

	// Description returns a human-readable description of the action for
	// logging purposes (e.g., "mark todo 123 as done").
	Description() string
}

// WriteStager provides write-staging capabilities to domain services.
// Domain services use this interface to stage entity updates (with their
// associated write actions) or to execute immediate actions.
//
// WriteStager is the domain's view of the application-layer RequestContext.
// Application services create the concrete implementation and pass it to
// domain services as this interface.
type WriteStager interface {
	// Stage updates the in-memory entity cache for the given key and queues
	// the associated action for later execution during Commit. Subsequent
	// reads for the same key (via GetOrFetch) will return the staged entity,
	// providing read-your-writes consistency within the request.
	Stage(key string, entity any, action Action) error

	// Execute runs an action immediately, independent of the commit queue.
	// The action is NOT added to the queue and will NOT be rolled back
	// during Commit rollback. Use for fire-and-forget or idempotent side
	// effects that should not participate in the transaction.
	Execute(action Action) error
}
