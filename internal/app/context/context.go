// Package appctx provides request-scoped context for orchestration services.
//
// RequestContext extends Go's context.Context with a thread-safe in-memory
// cache for memoized data fetching, optional per-entity shared references
// via SafeRef, and staged action execution with automatic rollback.
//
// A new RequestContext is created per HTTP request:
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
//
// All cache operations (GetOrFetch, GetRef, Put, Invalidate) are safe for
// concurrent use from multiple goroutines. The action queue (AddAction,
// AddGroup, Stage, Commit) is separately synchronized and independent of
// the cache lifecycle — cache operations work before, during, and after
// commit. See ADR-0002 for the full thread-safety design.
package appctx

import (
	"context"
	"errors"
	"fmt"
	"sync"

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

// ErrTypeMismatch is returned by GetOrFetch or GetRef when a cached value's
// type does not match the requested type T. This indicates a programming
// error where the same cache key is used with different types.
var ErrTypeMismatch = errors.New("appctx: cached value type mismatch")

// RequestContext is a request-scoped context wrapper providing a thread-safe
// in-memory cache and staged action execution. It embeds context.Context and
// adds memoization via GetOrFetch, shared mutable access via GetRef, and
// transactional action execution via Commit.
//
// The cache (protected by cacheMu) and the action queue (protected by
// queueMu) use independent mutexes so they do not constrain each other.
// Cache operations work regardless of commit state.
type RequestContext struct {
	context.Context

	// cacheMu protects cache and refs. All cache operations (GetOrFetch,
	// GetRef, Put, Invalidate) are safe for concurrent use.
	cacheMu sync.RWMutex
	cache   map[string]cacheEntry
	refs    map[string]any // map[string]*SafeRef[T] — per-entity shared refs

	// queueMu protects items and committed. The action queue lifecycle
	// (open → committed) is independent of the cache.
	queueMu   sync.Mutex
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
		refs:    make(map[string]any),
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
// GetOrFetch is safe for concurrent use. On a cache miss, the fetch happens
// without holding any lock. If two goroutines miss the same key, both fetch
// and the first to store wins (subsequent callers see the stored result).
func GetOrFetch[T any](rc *RequestContext, key string, fetchFn func(ctx context.Context) (T, error)) (T, error) {
	// Fast path: check cache under read lock.
	rc.cacheMu.RLock()
	if entry, ok := rc.cache[key]; ok {
		rc.cacheMu.RUnlock()
		return unwrapCacheEntry[T](key, entry)
	}
	rc.cacheMu.RUnlock()

	// Slow path: fetch without holding any lock.
	val, err := fetchFn(rc.Context)

	// Store with double-checked locking.
	rc.cacheMu.Lock()
	if entry, ok := rc.cache[key]; ok {
		// Another goroutine stored first — use their result.
		rc.cacheMu.Unlock()
		return unwrapCacheEntry[T](key, entry)
	}
	rc.cache[key] = cacheEntry{value: val, err: err}
	rc.cacheMu.Unlock()

	return val, err
}

// unwrapCacheEntry extracts a typed value from a cache entry, handling both
// raw values and SafeRef-wrapped values (from GetRef/Put).
func unwrapCacheEntry[T any](key string, entry cacheEntry) (T, error) {
	if entry.err != nil {
		var zero T
		return zero, entry.err
	}
	// Try SafeRef[T] first (entries upgraded by GetRef or Put).
	if ref, ok := entry.value.(*SafeRef[T]); ok {
		return ref.Get(), nil
	}
	// Try raw value (entries from GetOrFetch or Stage).
	v, ok := entry.value.(T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("%w: key %q holds %T, requested %T", ErrTypeMismatch, key, entry.value, zero)
	}
	return v, nil
}

// GetRef returns a thread-safe SafeRef for the given key, enabling multiple
// goroutines to read and write the same cached entity through a shared
// reference. If the key is not cached, fetchFn is called to populate it.
//
// All callers of GetRef for the same key receive the same SafeRef instance.
// Mutations via SafeRef.Set or SafeRef.Update are immediately visible to
// other goroutines holding the same reference.
//
// GetRef is safe for concurrent use. See GetOrFetch for cache-miss semantics.
func GetRef[T any](rc *RequestContext, key string, fetchFn func(ctx context.Context) (T, error)) (*SafeRef[T], error) {
	// Check refs map first (shared SafeRef instances).
	rc.cacheMu.RLock()
	if r, ok := rc.refs[key]; ok {
		rc.cacheMu.RUnlock()
		ref, ok := r.(*SafeRef[T])
		if !ok {
			return nil, fmt.Errorf("%w: key %q ref holds %T, requested *SafeRef[%T]",
				ErrTypeMismatch, key, r, *new(T))
		}
		return ref, nil
	}
	rc.cacheMu.RUnlock()

	// Ensure the underlying value is cached.
	val, err := GetOrFetch(rc, key, fetchFn)
	if err != nil {
		return nil, err
	}

	// Create or retrieve SafeRef under write lock.
	rc.cacheMu.Lock()
	if r, ok := rc.refs[key]; ok {
		// Another goroutine created the ref — use theirs.
		rc.cacheMu.Unlock()
		ref, ok := r.(*SafeRef[T])
		if !ok {
			return nil, fmt.Errorf("%w: key %q ref holds %T, requested *SafeRef[%T]",
				ErrTypeMismatch, key, r, *new(T))
		}
		return ref, nil
	}
	ref := NewRef(val)
	rc.refs[key] = ref
	// Also upgrade the cache entry so GetOrFetch sees SafeRef updates.
	rc.cache[key] = cacheEntry{value: ref}
	rc.cacheMu.Unlock()

	return ref, nil
}

// Put updates the cached value for the given key. If a SafeRef exists for
// the key (from a prior GetRef call), the SafeRef is updated so all holders
// see the new value. Otherwise a new cache entry is created.
//
// Use Put for write-through caching: after a goroutine performs a mutation
// via an API call, it calls Put to ensure subsequent reads (from any
// goroutine) see the updated entity.
//
// Put is safe for concurrent use.
func Put[T any](rc *RequestContext, key string, val T) {
	rc.cacheMu.Lock()
	defer rc.cacheMu.Unlock()

	// Update existing SafeRef if one exists.
	if r, ok := rc.refs[key]; ok {
		if ref, ok := r.(*SafeRef[T]); ok {
			ref.Set(val)
			return
		}
	}

	// No SafeRef — store raw value.
	rc.cache[key] = cacheEntry{value: val}
}

// Invalidate removes the cached value and any SafeRef for the given key.
// Subsequent GetOrFetch or GetRef calls will re-fetch from the source.
//
// Invalidate is safe for concurrent use. Goroutines holding a SafeRef
// obtained before invalidation retain their reference but it will no
// longer be returned by future GetRef calls.
func (rc *RequestContext) Invalidate(key string) {
	rc.cacheMu.Lock()
	delete(rc.cache, key)
	delete(rc.refs, key)
	rc.cacheMu.Unlock()
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
//
// Stage is safe for concurrent use. It acquires both cacheMu and queueMu.
func (rc *RequestContext) Stage(key string, entity any, action domain.Action) error {
	if action == nil {
		return ErrNilAction
	}

	rc.queueMu.Lock()
	defer rc.queueMu.Unlock()

	if rc.committed {
		return ErrAlreadyCommitted
	}

	rc.cacheMu.Lock()
	rc.cache[key] = cacheEntry{value: entity, err: nil}
	rc.cacheMu.Unlock()

	rc.items = append(rc.items, &singleAction{action: action})
	return nil
}

// contextKey is the unexported key type for storing RequestContext in context.
type contextKey struct{}

// WithRequestContext returns a new context with the given RequestContext
// stored in it. This is used by the appctx middleware to inject a
// RequestContext per HTTP request.
func WithRequestContext(ctx context.Context, rc *RequestContext) context.Context {
	return context.WithValue(ctx, contextKey{}, rc)
}

// FromContext extracts a *RequestContext from the context.
// Returns nil if no RequestContext is stored, allowing callers to fall back
// to direct calls when the middleware is not active (e.g., in unit tests).
func FromContext(ctx context.Context) *RequestContext {
	rc, _ := ctx.Value(contextKey{}).(*RequestContext)
	return rc
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
