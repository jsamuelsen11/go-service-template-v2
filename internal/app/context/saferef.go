package appctx

import "sync"

// SafeRef provides thread-safe concurrent access to a mutable entity.
// Multiple goroutines can safely read and write through the same reference.
//
// SafeRef uses a sync.RWMutex internally: reads (Get) acquire a shared read
// lock, while writes (Set, Update) acquire an exclusive write lock. This
// means concurrent reads do not block each other, while writes are serialized.
//
// Use Get for simple reads (returns a value copy), Set to replace the value,
// and Update for atomic in-place mutations.
type SafeRef[T any] struct {
	mu  sync.RWMutex
	val T
}

// NewRef creates a SafeRef initialized with the given value.
func NewRef[T any](val T) *SafeRef[T] {
	return &SafeRef[T]{val: val}
}

// Get returns a copy of the current value under a read lock.
// The returned value is safe to use without further synchronization.
func (r *SafeRef[T]) Get() T {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.val
}

// Set replaces the current value under a write lock.
func (r *SafeRef[T]) Set(val T) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.val = val
}

// Update applies fn to the value under a write lock, allowing atomic
// in-place mutations. The function receives a pointer to the value;
// modifications are visible to subsequent Get and Update calls.
func (r *SafeRef[T]) Update(fn func(*T)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	fn(&r.val)
}
