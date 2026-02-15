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
)

// ErrAlreadyCommitted is returned when AddAction, AddGroup, or Commit is
// called on a RequestContext that has already been committed.
var ErrAlreadyCommitted = errors.New("appctx: request context already committed")

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
// The same key must always be used with the same type T. Using different
// types for the same key results in a zero value on cache hit. Use
// DataProvider for type-safe, reusable fetch bindings.
//
// GetOrFetch is NOT safe for concurrent use. It is designed for sequential
// orchestration within a single request goroutine.
func GetOrFetch[T any](rc *RequestContext, key string, fetchFn func(ctx context.Context) (T, error)) (T, error) {
	if entry, ok := rc.cache[key]; ok {
		v, _ := entry.value.(T)
		return v, entry.err
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
