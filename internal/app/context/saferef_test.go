package appctx_test

import (
	"sync"
	"sync/atomic"
	"testing"

	appctx "github.com/jsamuelsen11/go-service-template-v2/internal/app/context"
)

func TestSafeRef_GetSet(t *testing.T) {
	t.Parallel()

	ref := appctx.NewRef("initial")

	if got := ref.Get(); got != "initial" {
		t.Fatalf("Get() = %q, want %q", got, "initial")
	}

	ref.Set("updated")

	if got := ref.Get(); got != "updated" {
		t.Fatalf("Get() = %q, want %q", got, "updated")
	}
}

func TestSafeRef_Update(t *testing.T) {
	t.Parallel()

	type entity struct {
		Name  string
		Count int
	}

	ref := appctx.NewRef(entity{Name: "test", Count: 0})

	ref.Update(func(e *entity) {
		e.Count = 42
	})

	got := ref.Get()
	if got.Name != "test" || got.Count != 42 {
		t.Fatalf("after Update: got %+v, want {Name:test Count:42}", got)
	}
}

func TestSafeRef_ConcurrentReads(t *testing.T) {
	t.Parallel()

	ref := appctx.NewRef(99)

	const goroutines = 50
	var wg sync.WaitGroup

	for range goroutines {
		wg.Go(func() {
			if got := ref.Get(); got != 99 {
				t.Errorf("Get() = %d, want 99", got)
			}
		})
	}

	wg.Wait()
}

func TestSafeRef_ConcurrentReadWrite(t *testing.T) {
	t.Parallel()

	ref := appctx.NewRef(0)

	const writers = 10
	const readers = 20
	var wg sync.WaitGroup

	// Writers increment the value.
	for range writers {
		wg.Go(func() {
			ref.Update(func(v *int) { *v++ })
		})
	}

	// Readers just call Get — should never panic or race.
	for range readers {
		wg.Go(func() {
			_ = ref.Get()
		})
	}

	wg.Wait()

	if got := ref.Get(); got != writers {
		t.Errorf("final value = %d, want %d", got, writers)
	}
}

func TestSafeRef_ConcurrentUpdates(t *testing.T) {
	t.Parallel()

	ref := appctx.NewRef(int64(0))

	const goroutines = 100
	var wg sync.WaitGroup
	var expected atomic.Int64

	for range goroutines {
		wg.Go(func() {
			expected.Add(1)
			ref.Update(func(v *int64) { *v++ })
		})
	}

	wg.Wait()

	if got := ref.Get(); got != expected.Load() {
		t.Errorf("final value = %d, want %d", got, expected.Load())
	}
}
