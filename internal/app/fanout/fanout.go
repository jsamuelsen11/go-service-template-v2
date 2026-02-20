// Package fanout provides a generic, bounded-concurrency fan-out helper for
// application-layer orchestration. It runs a function across a slice of items
// using a fixed number of worker goroutines, preserving input order in results.
//
// The helper is intentionally minimal: it manages goroutines, bounded
// concurrency via a semaphore channel, and context cancellation. It has no
// dependencies beyond the standard library, keeping it reusable across
// entities and services.
package fanout

import (
	"context"
	"sync"
)

// Result holds the outcome of processing a single item.
// Either Value is populated (on success) or Err is non-nil (on failure).
type Result[R any] struct {
	Value R
	Err   error
}

// Run executes fn for each item in items using at most maxWorkers concurrent
// goroutines. Results are returned in the same order as the input items.
//
// If ctx is canceled while a goroutine is waiting for a semaphore slot,
// that goroutine records ctx.Err() and does not call fn. Goroutines that
// have already acquired a slot run to completion (fn is responsible for
// checking ctx internally if it supports cancellation).
//
// Run blocks until all goroutines complete. If items is empty, it returns
// an empty non-nil slice immediately.
//
// maxWorkers must be >= 1. If maxWorkers >= len(items), all items run
// concurrently with no semaphore contention.
func Run[T, R any](ctx context.Context, maxWorkers int, items []T, fn func(context.Context, T) (R, error)) []Result[R] {
	if len(items) == 0 {
		return []Result[R]{}
	}

	results := make([]Result[R], len(items))
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	for i, item := range items {
		wg.Add(1)
		go func(idx int, it T) {
			defer wg.Done()

			// Context-aware semaphore acquisition.
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				results[idx] = Result[R]{Err: ctx.Err()}
				return
			}

			val, err := fn(ctx, it)
			results[idx] = Result[R]{Value: val, Err: err}
		}(i, item)
	}

	wg.Wait()
	return results
}
