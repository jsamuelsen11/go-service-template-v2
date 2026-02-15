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
// Returns ErrAlreadyCommitted if called more than once.
func (rc *RequestContext) Commit(ctx context.Context) error {
	if rc.committed {
		return ErrAlreadyCommitted
	}
	rc.committed = true

	logger := logging.FromContext(ctx)

	for i, item := range rc.items {
		logger.InfoContext(ctx, "executing action",
			slog.String("operation", "RequestContext.Commit"),
			slog.Int("step", i+1),
			slog.Int("total", len(rc.items)),
			slog.String("action", item.description()),
		)

		if err := item.execute(ctx); err != nil {
			logger.ErrorContext(ctx, "action failed, initiating rollback",
				slog.String("operation", "RequestContext.Commit"),
				slog.Int("failed_step", i+1),
				slog.String("action", item.description()),
				slog.Any("error", err),
			)
			rc.rollbackItems(ctx, i-1, logger)
			return fmt.Errorf("executing %s: %w", item.description(), err)
		}
	}

	return nil
}

// rollbackItems rolls back items 0..upTo (inclusive) in reverse order.
// Rollback errors are logged at ERROR level and do not stop the rollback
// of remaining items.
func (rc *RequestContext) rollbackItems(ctx context.Context, upTo int, logger *slog.Logger) {
	for i := upTo; i >= 0; i-- {
		item := rc.items[i]

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
