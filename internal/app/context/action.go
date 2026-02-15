package appctx

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jsamuelsen11/go-service-template-v2/internal/platform/logging"
)

// Action represents a single executable operation with rollback capability.
// Implementations should be idempotent where possible to support safe retries.
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

// actionItem is the internal interface for executable items in the action
// queue. Both single actions and action groups implement this interface.
type actionItem interface {
	execute(ctx context.Context) error
	rollback(ctx context.Context) error
	description() string
}

// singleAction wraps an Action to satisfy the actionItem interface.
type singleAction struct {
	action Action
}

func (s *singleAction) execute(ctx context.Context) error  { return s.action.Execute(ctx) }
func (s *singleAction) rollback(ctx context.Context) error { return s.action.Rollback(ctx) }
func (s *singleAction) description() string                { return s.action.Description() }

// actionGroup holds multiple actions that execute in parallel. If any action
// fails, in-progress actions are canceled via context and successfully
// completed actions are rolled back in reverse insertion order.
type actionGroup struct {
	actions   []Action
	completed []Action
}

func (g *actionGroup) execute(ctx context.Context) error {
	if len(g.actions) == 0 {
		return nil
	}

	groupCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	type result struct {
		index int
		err   error
	}

	results := make(chan result, len(g.actions))

	for i, action := range g.actions {
		go func(idx int, a Action) {
			results <- result{index: idx, err: a.Execute(groupCtx)}
		}(i, action)
	}

	completedSet := make([]bool, len(g.actions))
	var firstErr error

	for range g.actions {
		r := <-results
		if r.err != nil {
			if firstErr == nil {
				firstErr = r.err
				cancel()
			}
		} else {
			completedSet[r.index] = true
		}
	}

	g.completed = nil
	for i, done := range completedSet {
		if done {
			g.completed = append(g.completed, g.actions[i])
		}
	}

	if firstErr != nil {
		g.rollbackCompleted(ctx)
		return firstErr
	}

	return nil
}

func (g *actionGroup) rollback(ctx context.Context) error {
	g.rollbackCompleted(ctx)
	return nil
}

// rollbackCompleted rolls back successfully completed actions in reverse
// insertion order. Rollback errors are logged but do not stop the rollback
// of remaining actions.
func (g *actionGroup) rollbackCompleted(ctx context.Context) {
	logger := logging.FromContext(ctx)
	for i := len(g.completed) - 1; i >= 0; i-- {
		action := g.completed[i]
		if err := action.Rollback(ctx); err != nil {
			logger.ErrorContext(ctx, "rollback failed in action group",
				slog.String("operation", "ActionGroup.rollback"),
				slog.String("action", action.Description()),
				slog.Any("error", err),
			)
		}
	}
}

func (g *actionGroup) description() string {
	switch len(g.actions) {
	case 0:
		return "empty action group"
	case 1:
		return g.actions[0].Description()
	default:
		return fmt.Sprintf("action group (%d actions: %s, ...)", len(g.actions), g.actions[0].Description())
	}
}

// AddAction stages a single action for later execution by Commit.
// Returns ErrNilAction if action is nil, or ErrAlreadyCommitted if the
// RequestContext has already been committed.
func (rc *RequestContext) AddAction(action Action) error {
	if action == nil {
		return ErrNilAction
	}
	if rc.committed {
		return ErrAlreadyCommitted
	}
	rc.items = append(rc.items, &singleAction{action: action})
	return nil
}

// AddGroup stages an action group for parallel execution by Commit.
// All actions in the group execute concurrently when the group's turn
// arrives during Commit. Returns ErrNilAction if any action is nil, or
// ErrAlreadyCommitted if the RequestContext has already been committed.
func (rc *RequestContext) AddGroup(actions ...Action) error {
	for _, a := range actions {
		if a == nil {
			return ErrNilAction
		}
	}
	if rc.committed {
		return ErrAlreadyCommitted
	}
	rc.items = append(rc.items, &actionGroup{actions: actions})
	return nil
}
