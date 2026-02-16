// Package appctx provides request-scoped context for orchestration services.
//
// RequestContext extends Go's context.Context with in-memory caching for
// memoized data fetching and staged action execution with automatic rollback.
//
// A new RequestContext is created per HTTP request and must not be shared
// between concurrent requests:
//
//	rc := appctx.New(ctx)
//
//	// Stage 1: Fetch data with memoization
//	todo, err := appctx.GetOrFetch(rc, "todo:123", fetchTodo)
//
//	// Stage 2: Stage write operations
//	rc.AddAction(&MarkDoneAction{TodoID: 123})
//
//	// Stage 3: Execute all staged actions
//	err = rc.Commit(ctx)
package appctx

import (
	"context"
	"errors"
	"fmt"

	"github.com/jsamuelsen11/go-service-template-v2/internal/domain"
)

// Compile-time check that RequestContext implements domain.WriteStager.
var _ domain.WriteStager = (*RequestContext)(nil)

// ErrAlreadyCommitted is returned when AddAction, AddGroup, or Commit is
// called on a RequestContext that has already been committed.
var ErrAlreadyCommitted = errors.New("appctx: request context already committed")

// ErrNilAction is returned when a nil Action is passed to AddAction or
// AddGroup.
var ErrNilAction = errors.New("appctx: nil action")

// ErrTypeMismatch is returned by GetOrFetch when a cached value's type does
// not match the requested type T. This indicates a programming error where
// the same cache key is used with different types.
var ErrTypeMismatch = errors.New("appctx: cached value type mismatch")

// RequestContext is a request-scoped context wrapper providing in-memory
// caching and staged action execution. It embeds context.Context and adds
// memoization via GetOrFetch and transactional action execution via Commit.
//
// A RequestContext is strictly request-scoped: create a new instance for each
// HTTP request. It is NOT safe for concurrent use from multiple goroutines.
type RequestContext struct {
	context.Context
	cache     map[string]cacheEntry
	items     []actionItem
	committed bool
}

// cacheEntry stores the result of a GetOrFetch call, including any error.
// Both successful results and errors are cached to prevent redundant calls
// within the same request.
type cacheEntry struct {
	value any
	err   error
}

// New creates a RequestContext wrapping the given context.Context.
// The returned RequestContext has an empty cache and no staged actions.
func New(ctx context.Context) *RequestContext {
	return &RequestContext{
		Context: ctx,
		cache:   make(map[string]cacheEntry),
	}
}

// GetOrFetch returns a cached value for the given key, or calls fetchFn to
// fetch and cache it. Both successful results and errors are cached to
// prevent redundant calls within the same request.
//
// The same key must always be used with the same type T. If a cached value
// exists but its type does not match T, GetOrFetch returns ErrTypeMismatch.
// Use DataProvider for type-safe, reusable fetch bindings that prevent this.
//
// GetOrFetch is NOT safe for concurrent use. It is designed for sequential
// orchestration within a single request goroutine.
func GetOrFetch[T any](rc *RequestContext, key string, fetchFn func(ctx context.Context) (T, error)) (T, error) {
	if entry, ok := rc.cache[key]; ok {
		if entry.err != nil {
			var zero T
			return zero, entry.err
		}
		v, ok := entry.value.(T)
		if !ok {
			var zero T
			return zero, fmt.Errorf("%w: key %q holds %T, requested %T", ErrTypeMismatch, key, entry.value, zero)
		}
		return v, nil
	}

	val, err := fetchFn(rc.Context)
	rc.cache[key] = cacheEntry{value: val, err: err}
	return val, err
}

// DataProvider is a type-safe wrapper around GetOrFetch for a specific data
// type. It binds a cache key and fetch function together, allowing callers
// to retrieve data without specifying the key and function each time.
type DataProvider[T any] struct {
	key     string
	fetchFn func(ctx context.Context) (T, error)
}

// NewDataProvider creates a DataProvider with the given cache key and fetch
// function.
func NewDataProvider[T any](key string, fetchFn func(ctx context.Context) (T, error)) *DataProvider[T] {
	return &DataProvider[T]{key: key, fetchFn: fetchFn}
}

// Get returns the cached value or fetches it using the provider's fetch
// function. Equivalent to calling GetOrFetch with the provider's key and
// fetch function.
func (p *DataProvider[T]) Get(rc *RequestContext) (T, error) {
	return GetOrFetch(rc, p.key, p.fetchFn)
}

// Stage updates the in-memory cache for the given key with the provided
// entity and queues the action for execution during Commit. This provides
// read-your-writes consistency: subsequent GetOrFetch calls for the same
// key will return the staged entity rather than re-fetching.
//
// Returns ErrNilAction if action is nil, or ErrAlreadyCommitted if the
// RequestContext has already been committed.
func (rc *RequestContext) Stage(key string, entity any, action domain.Action) error {
	if action == nil {
		return ErrNilAction
	}
	if rc.committed {
		return ErrAlreadyCommitted
	}
	rc.cache[key] = cacheEntry{value: entity, err: nil}
	rc.items = append(rc.items, &singleAction{action: action})
	return nil
}

// Execute runs an action immediately, independent of the commit queue.
// The action is NOT added to the staged items and will NOT participate
// in Commit's execution or rollback sequence.
//
// Execute uses the RequestContext's embedded context for the action.
// Returns ErrNilAction if action is nil. Unlike Stage and AddAction,
// Execute works after the RequestContext has been committed, since it
// is independent of the queue.
func (rc *RequestContext) Execute(action domain.Action) error {
	if action == nil {
		return ErrNilAction
	}
	return action.Execute(rc.Context)
}
