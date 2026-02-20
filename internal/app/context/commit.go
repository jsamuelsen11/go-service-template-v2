package appctx

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jsamuelsen11/go-service-template-v2/internal/platform/logging"
)

// Commit executes all staged actions and action groups in insertion order.
// If any item fails, previously completed items are rolled back in reverse
// order. Rollback errors are logged but do not affect the returned error.
//
// After Commit returns (whether success or failure), the RequestContext is
// marked as committed and no further actions can be staged.
//
// Commit is safe for concurrent use but should only be called once per
// request. The committed flag and items snapshot are captured under lock;
// action execution happens without holding any lock.
//
// Returns ErrAlreadyCommitted if called more than once.
func (rc *RequestContext) Commit(ctx context.Context) error {
	rc.queueMu.Lock()
	if rc.committed {
		rc.queueMu.Unlock()
		return ErrAlreadyCommitted
	}
	rc.committed = true
	// Snapshot items under lock. Once committed=true, no goroutine can
	// append to rc.items via AddAction/AddGroup/Stage, so iterating the
	// snapshot without holding the lock is safe.
	items := rc.items
	rc.queueMu.Unlock()

	logger := logging.FromContext(ctx)

	for i, item := range items {
		logger.InfoContext(ctx, "executing action",
			slog.String("operation", "RequestContext.Commit"),
			slog.Int("step", i+1),
			slog.Int("total", len(items)),
			slog.String("action", item.description()),
		)

		if err := item.execute(ctx); err != nil {
			logger.ErrorContext(ctx, "action failed, initiating rollback",
				slog.String("operation", "RequestContext.Commit"),
				slog.Int("failed_step", i+1),
				slog.String("action", item.description()),
				slog.Any("error", err),
			)
			rollbackItems(ctx, items, i-1, logger)
			return fmt.Errorf("executing %s: %w", item.description(), err)
		}
	}

	return nil
}

// rollbackItems rolls back items 0..upTo (inclusive) in reverse order.
// Rollback errors are logged at ERROR level and do not stop the rollback
// of remaining items.
func rollbackItems(ctx context.Context, items []actionItem, upTo int, logger *slog.Logger) {
	for i := upTo; i >= 0; i-- {
		item := items[i]

		logger.InfoContext(ctx, "rolling back action",
			slog.String("operation", "RequestContext.Commit"),
			slog.Int("step", i+1),
			slog.String("action", item.description()),
		)

		if err := item.rollback(ctx); err != nil {
			logger.ErrorContext(ctx, "rollback failed",
				slog.String("operation", "RequestContext.Commit"),
				slog.Int("step", i+1),
				slog.String("action", item.description()),
				slog.Any("error", err),
			)
		}
	}
}
